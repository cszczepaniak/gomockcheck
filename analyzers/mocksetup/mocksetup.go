package mocksetup

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"iter"
	"strconv"
	"strings"
	"sync"

	"github.com/cszczepaniak/gomockcheck/analyzers/internal/names"
	"github.com/cszczepaniak/gomockcheck/analyzers/internal/typeutils"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

func New() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "mocksetup",
		Doc:      "Checks for common mock setup mistakes",
		Run:      run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

var (
	mockType              types.Type
	mockMethodSet         *types.MethodSet
	unexportedMockMethods map[string]struct{}
	initOnce              sync.Once
)

func setMockType(typ types.Type) {
	initOnce.Do(func() {
		mockType = typ
		mockMethodSet = types.NewMethodSet(typ)

		unexportedMockMethods = make(map[string]struct{})
		for m := range iterMethodSet(mockMethodSet) {
			if !m.Obj().Exported() {
				unexportedMockMethods[m.Obj().Name()] = struct{}{}
			}
		}
	})
}

func run(pass *analysis.Pass) (any, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	for n := range inspector.PreorderSeq(&ast.CallExpr{}) {
		mockDotOnCall := n.(*ast.CallExpr)

		if !isMockDotOn(pass.TypesInfo, mockDotOnCall) {
			continue
		}

		if !checkMockDotOnCall(pass, mockDotOnCall) {
			continue
		}

		// TODO check more things:
		// - If the mocked call has return arguments, there must be a .Return
		// - There must be the correct number of arguments to .Return
		// - The arguments to .Return must be of the correct type
	}

	return nil, nil
}

func isMockDotOn(typesInfo *types.Info, c *ast.CallExpr) bool {
	fn := typeutil.StaticCallee(typesInfo, c)
	fnSig := fn.Signature()
	recv := fnSig.Recv()
	if recv == nil || fn.Name() != "On" {
		return false
	}

	obj := typeutils.GetObjForPtrToNamedType(recv.Type())
	if obj == nil {
		return false
	}

	if obj.Pkg().Path() != "github.com/stretchr/testify/mock" || obj.Name() != "Mock" {
		return false
	}

	// We've found the mock type. Let's notice its method set once so we don't have to keep
	// recomputing this.
	setMockType(recv.Type())
	return true
}

func checkMockDotOnCall(pass *analysis.Pass, mockDotOnCall *ast.CallExpr) bool {
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
	for m := range distinctMethods(pass.Pkg, selTyp.Recv()) {
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

	var badIdxStrs []string
	for i, arg := range mockDotOnCall.Args[1:] {
		want := sig.Params().At(i).Type()
		if !argTypeIsValid(pass.TypesInfo, want, arg) {
			badIdxStrs = append(badIdxStrs, strconv.Itoa(i))
		}
	}

	if len(badIdxStrs) > 0 {
		pass.Reportf(
			mockDotOnCall.Pos(),
			"parameter indexes [%s] had incorrect types",
			strings.Join(badIdxStrs, ", "),
		)
	}

	return true
}

func argTypeIsValid(info *types.Info, want types.Type, got ast.Expr) bool {
	if isMockAnything(info, got) {
		return true
	}

	gotTyp, ok := info.Types[got]
	if !ok {
		// TODO: what?
		return false
	}

	return types.Identical(want, gotTyp.Type)
}

func isMockAnything(info *types.Info, arg ast.Expr) bool {
	var obj types.Object
	switch arg := arg.(type) {
	case *ast.Ident:
		obj = info.ObjectOf(arg)
	case *ast.SelectorExpr:
		obj = info.ObjectOf(arg.Sel)
	}

	if obj != nil {
		fmt.Println(obj)
	}
	return names.IsTestifyPkg(obj) && obj.Name() == "Anything"
}

// distinctMethods returns the methods on this type that aren't on the mock type. Precondition:
// setMockType has been called.
func distinctMethods(pkg *types.Package, typ types.Type) iter.Seq[*types.Selection] {
	return func(yield func(*types.Selection) bool) {
		mSet := types.NewMethodSet(typ)
		for i := range mSet.Len() {
			method := mSet.At(i)
			mockMethod := mockMethodSet.Lookup(pkg, method.Obj().Name())

			_, isUnexpectedMockMethod := unexportedMockMethods[method.Obj().Name()]

			if mockMethod == nil && !isUnexpectedMockMethod {
				if !yield(method) {
					return
				}
			}
		}
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
