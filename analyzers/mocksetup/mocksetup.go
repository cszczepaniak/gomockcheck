package mocksetup

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"iter"
	"slices"

	"github.com/cszczepaniak/gomockcheck/analyzers/internal/typeutils"
	"github.com/cszczepaniak/gomockcheck/names"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

func New(typs ...names.QualifiedType) *analysis.Analyzer {
	r := &runner{
		types: slices.Concat([]names.QualifiedType{{
			PkgPath: "github.com/stretchr/testify/mock",
			Name:    "Mock",
		}}, typs),
	}

	return &analysis.Analyzer{
		Name:     "mocksetup",
		Doc:      "Checks for common mock setup mistakes",
		Run:      r.run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

type runner struct {
	types []names.QualifiedType
}

func (r *runner) run(pass *analysis.Pass) (any, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspector.WithStack(
		[]ast.Node{&ast.CallExpr{}},
		func(n ast.Node, push bool, stack []ast.Node) (proceed bool) {
			if !push {
				return false
			}

			mockDotOnCall := n.(*ast.CallExpr)
			if !r.isMockDotOn(pass.TypesInfo, mockDotOnCall) {
				return true
			}

			if !r.checkMockDotOnCall(pass, mockDotOnCall) {
				return true
			}

			// TODO check more things:
			// - If the mocked call has return arguments, there must be a .Return
			// - There must be the correct number of arguments to .Return
			// - The arguments to .Return must be of the correct type
			return true
		},
	)

	return nil, nil
}

func (r *runner) isMockDotOn(typesInfo *types.Info, c *ast.CallExpr) bool {
	fn := typeutil.StaticCallee(typesInfo, c)
	if fn == nil {
		return false
	}

	fnSig := fn.Signature()
	recv := fnSig.Recv()
	if recv == nil || fn.Name() != "On" {
		return false
	}

	obj := typeutils.GetObjForPtrToNamedType(recv.Type())
	if obj == nil {
		return false
	}

	return names.IsOneOf(obj, r.types...)
}

func (r *runner) checkMockDotOnCall(pass *analysis.Pass, mockDotOnCall *ast.CallExpr) bool {
	// We know this function has at least one argument and that the first one is always a
	// string. Let's check to make sure it's a constant and report a problem if it isn't.
	typ, ok := pass.TypesInfo.Types[mockDotOnCall.Args[0]]
	if !ok {
		// This would be weird, right?
		return false
	}

	if typ.Value == nil {
		// We won't have a value if the argument isn't const. Let's report this.
		pass.Reportf(mockDotOnCall.Args[0].Pos(), "the name of a mocked method should be a constant")
		return false
	}

	mockedMethodName := constant.StringVal(typ.Value)

	sel, ok := mockDotOnCall.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	selTyp, ok := pass.TypesInfo.Selections[sel]
	if !ok {
		return false
	}

	var mockedMethod *types.Selection
	for m := range r.distinctMethods(pass.Pkg, selTyp.Recv()) {
		if m.Obj().Name() == mockedMethodName {
			mockedMethod = m
			break
		}
	}

	if mockedMethod == nil {
		pass.Reportf(mockDotOnCall.Args[0].Pos(), "%q is not a method of %s", mockedMethodName, selTyp.Recv())
		return false
	}

	// Exclude the method name from the number of args supplied to the mock setup.
	numMockedArgs := len(mockDotOnCall.Args) - 1

	sig := mockedMethod.Type().(*types.Signature)
	if sig.Params().Len() != numMockedArgs {
		pass.Reportf(
			mockDotOnCall.Pos(),
			"call is mocked for %d arguments, but method %q takes %d",
			numMockedArgs,
			mockedMethodName,
			sig.Params().Len(),
		)
		return false
	}

	for i, arg := range mockDotOnCall.Args[1:] {
		want := sig.Params().At(i).Type()

		switch {
		case isMockAnything(pass.TypesInfo, arg):
			continue
		case handleMockAnythingOfType(pass, want, arg):
			continue
		case handleMockMatchedBy(pass, want, arg):
			continue
		}

		argTyp := pass.TypesInfo.TypeOf(arg)
		if !types.AssignableTo(argTyp, want) {
			msg := fmt.Sprintf("invalid parameter type in mock setup; %s is not assignable to %s", argTyp, want)
			if i == len(mockDotOnCall.Args)-2 &&
				// If we wanted []T but had T for the variadic parameter we'll add more help.
				sig.Variadic() &&
				types.Identical(want, types.NewSlice(pass.TypesInfo.TypeOf(arg))) {
				msg += " (hint: last parameter is variadic, make it a slice)"
			}

			pass.Reportf(arg.Pos(), msg)
		}
	}

	return true
}

func isMockAnything(info *types.Info, arg ast.Expr) bool {
	var obj types.Object
	switch arg := arg.(type) {
	case *ast.Ident:
		obj = info.ObjectOf(arg)
	case *ast.SelectorExpr:
		obj = info.ObjectOf(arg.Sel)
	}

	return names.IsTestifySymbol(obj, "Anything")
}

func handleMockAnythingOfType(pass *analysis.Pass, want types.Type, arg ast.Expr) bool {
	call, ok := arg.(*ast.CallExpr)
	if !ok {
		return false
	}

	callee := typeutil.StaticCallee(pass.TypesInfo, call)
	if !names.IsTestifySymbol(callee, "AnythingOfType") {
		return false
	}

	// If the actual type is an interface, the AnythingOfType is likely asserting that the type
	// passed in is a specific implementation of that interface. We can't do much more to verify,
	// but we should consider it "handled" because this was a mock.AnythingOfType.
	if getInterfaceType(want) != nil {
		return true
	}

	// Uses of mock.AnythingOfType with concrete types are not allowed. These should be simplified
	// to mock.Anything. Use the same import name that was used for the mock.AnythingOfType
	var edit string
	switch call := call.Fun.(type) {
	case *ast.Ident:
		edit = "Anything"
	case *ast.SelectorExpr:
		edit = call.X.(*ast.Ident).Name + ".Anything"
	}

	var suggestedFixes []analysis.SuggestedFix
	if edit != "" {
		suggestedFixes = []analysis.SuggestedFix{{
			Message: "replace with mock.Anything",
			TextEdits: []analysis.TextEdit{{
				Pos:     arg.Pos(),
				End:     arg.End(),
				NewText: []byte(edit),
			}},
		}}
	}

	pass.Report(analysis.Diagnostic{
		Pos:            arg.Pos(),
		End:            arg.End(),
		Message:        "mock.AnythingOfType is equivalent to mock.Anything when the input type is concrete; use mock.Anything instead",
		SuggestedFixes: suggestedFixes,
	})

	return true
}

func handleMockMatchedBy(pass *analysis.Pass, want types.Type, arg ast.Expr) bool {
	call, ok := arg.(*ast.CallExpr)
	if !ok {
		return false
	}

	callee := typeutil.StaticCallee(pass.TypesInfo, call)
	if !names.IsTestifySymbol(callee, "MatchedBy") {
		return false
	}

	report := func() {
		pass.Reportf(call.Args[0].Pos(), "the argument to mock.MatchedBy must be func(%s) bool", want)
	}

	argTyp := pass.TypesInfo.TypeOf(call.Args[0])
	fn, ok := argTyp.(*types.Signature)
	if !ok {
		report()
		return true
	}

	boolTyp := types.Universe.Lookup("bool").Type()
	if fn.Params().Len() != 1 ||
		!types.Identical(want, fn.Params().At(0).Type()) ||
		fn.Results().Len() != 1 ||
		!types.Identical(boolTyp, fn.Results().At(0).Type()) {
		report()
	}

	// Return true because at this point we've seen a properly configured mock.MatchedBy
	return true
}

func getInterfaceType(typ types.Type) *types.Interface {
	switch typ := typ.(type) {
	case *types.Interface:
		return typ
	case *types.Named:
		return getInterfaceType(typ.Underlying())
	default:
		return nil
	}
}

// distinctMethods returns the methods on this type that aren't on the mock type. Precondition:
// setMockType has been called.
func (r *runner) distinctMethods(pkg *types.Package, typ types.Type) iter.Seq[*types.Selection] {
	mockTyp := r.getEmbeddedMockType(typ)
	mockMethodSet := types.NewMethodSet(mockTyp)
	unexportedMockMethods := make(map[string]struct{})
	for m := range iterMethodSet(mockMethodSet) {
		if !m.Obj().Exported() {
			unexportedMockMethods[m.Obj().Name()] = struct{}{}
		}
	}

	return func(yield func(*types.Selection) bool) {
		mSet := types.NewMethodSet(typ)
		for method := range iterMethodSet(mSet) {
			_, isUnexpectedMockMethod := unexportedMockMethods[method.Obj().Name()]
			if isUnexpectedMockMethod {
				continue
			}

			mockMethod := mockMethodSet.Lookup(pkg, method.Obj().Name())
			if mockMethod == nil && !yield(method) {
				return
			}
		}
	}
}

func (r *runner) getEmbeddedMockType(typ types.Type) types.Type {
	s := getStructType(typ)
	for i := range s.NumFields() {
		field := s.Field(i)
		if !field.Embedded() {
			continue
		}

		named, ok := field.Type().(*types.Named)
		if !ok {
			continue
		}

		if names.IsOneOf(named.Obj(), r.types...) {
			return named
		}
	}
	return nil
}

func getStructType(typ types.Type) *types.Struct {
	switch typ := typ.(type) {
	case *types.Struct:
		return typ
	case *types.Named:
		return getStructType(typ.Underlying())
	case *types.Pointer:
		return getStructType(typ.Elem())
	default:
		return nil
	}
}

func iterMethodSet(mSet *types.MethodSet) iter.Seq[*types.Selection] {
	return func(yield func(*types.Selection) bool) {
		for i := range mSet.Len() {
			if !yield(mSet.At(i)) {
				return
			}
		}
	}
}
