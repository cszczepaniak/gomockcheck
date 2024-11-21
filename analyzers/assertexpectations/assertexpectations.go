package assertexpectations

import (
	"fmt"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"github.com/cszczepaniak/gomockcheck/analyzers/internal/typeutils"
	"github.com/cszczepaniak/gomockcheck/names"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

func New(typs ...names.QualifiedType) *analysis.Analyzer {
	r := runner{
		types: slices.Concat([]names.QualifiedType{{
			PkgPath: "github.com/stretchr/testify/mock",
			Name:    "Mock",
		}}, typs),
	}

	return &analysis.Analyzer{
		Name:     "assertexpectations",
		Doc:      "Ensure that AssertExpectations is called on mock objects before they're used",
		Run:      r.run,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	}
}

var debug = false

func setDebug(val bool) {
	debug = val
}

func debugf(s string, args ...any) {
	if debug {
		if !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		fmt.Printf(s, args...)
	}
}

type runner struct {
	types []names.QualifiedType
}

func (r runner) isMockObj(obj types.Object) bool {
	if obj == nil {
		return false
	}

	return names.IsOneOf(obj, r.types...)
}

func (r runner) run(pass *analysis.Pass) (any, error) {
	pssa := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	for _, f := range pssa.SrcFuncs {
		for _, b := range f.Blocks {
			if b == f.Recover {
				continue
			}

			for _, instr := range b.Instrs {
				alloc, ok := instr.(*ssa.Alloc)
				if !ok {
					continue
				}

				if !r.hasEmbeddedMockType(alloc.Type()) {
					continue
				}

				r.handleReferrers(pass, alloc, f.Recover)
			}
		}
	}

	return nil, nil
}

func (r runner) handleReferrers(pass *analysis.Pass, alloc *ssa.Alloc, skipBlock *ssa.BasicBlock) {
	var pos token.Pos
	for _, ref := range *alloc.Referrers() {
		// It's possible that an alloc from one block will refer to the recover block. We don't want
		// to analyze things inside of the recover block.
		if ref.Block() == skipBlock {
			continue
		}
		continuation := r.handleReferrer(alloc, ref)
		switch c := continuation.(type) {
		case keepGoing:
			if pos == 0 {
				pos = c.pos
			}
		case report:
			if pos == 0 {
				pos = alloc.Pos()
			}
			pass.Reportf(
				pos,
				"mocks must have an AssertExpectations registered in a defer or t.Cleanup",
			)
			return
		case succeed:
			return
		}
	}
}

type continuation interface{ c() }

type report struct{}

func (report) c() {}

type succeed struct{}

func (succeed) c() {}

type keepGoing struct {
	pos token.Pos
}

func (keepGoing) c() {}

func (r runner) handleReferrer(alloc *ssa.Alloc, instr ssa.Instruction) continuation {
	switch ref := instr.(type) {
	case *ssa.Store:
		if ref.Addr != alloc {
			return keepGoing{}
		}

		_, ok := ref.Val.(*ssa.Alloc)
		if ok {
			// If the RHS of the store operation is an allocation, that means it's a struct literal
			// and we should analyze it. Let's include the position of the value because there are
			// some cases that the position of the Addr here points to an unintuitive location.
			return keepGoing{pos: ref.Val.Pos()}
		} else {
			// If it's not an allocation it could be something like a function call. If we're
			// storing the result of a function call, we'll analyze that function elsewhere so we
			// can assume it has set up AssertExpectations correctly.
			return succeed{}
		}
	case *ssa.MakeClosure:
		// This is the case that we're referring to the mock in a closure. We'll check to see if
		// this is a closure passed into a t.Cleanup or a closure in a defer, and if that closure
		// calls AssertExpectations.
		isCleanup := false
		for _, ref := range *ref.Referrers() {
			isCleanup = isTCleanupOrDefer(ref)
			if isCleanup {
				break
			}
		}

		// If it's not a cleanup function, we should report because we're using the mock before setting up
		// the defer or cleanup.
		if !isCleanup {
			return report{}
		}

		// We're closing over the mock, so it'll be one of the free variables in the closure. We'll find it
		// and then look through what refers to it to see if we end up calling AssertExpectations.
		var freeVar *ssa.FreeVar
		closure := ref.Fn.(*ssa.Function)
		for i, b := range ref.Bindings {
			if b == alloc {
				freeVar = closure.FreeVars[i]
				break
			}
		}

		for _, ref := range *freeVar.Referrers() {
			val, ok := ref.(ssa.Value)
			if !ok {
				continue
			}

			// Currently we only enforce that there's an AssertExpectations call somewhere in the cleanup
			// function, but we could maybe also additionally force it to be the first call.
			c := resultantCall(val)
			if c != nil && r.isAssertExpectations(c.Call) {
				return succeed{}
			}
		}

		return report{}

	case ssa.Value:
		// We allow calling mock.Test(t) before setting up AssertExpectations; this is fine to do
		// and they can be done in either order.
		c := resultantCall(ref)
		if c != nil && r.isTest(c.Call) {
			return keepGoing{}
		}

		// Check to see if this value is referred to in a defer statement, which we allow. The defer
		// must be in a top-level test function.
		deferredCall, ok := deferredCall(ref)
		if !ok || !r.isAssertExpectations(deferredCall) {
			return report{}
		}

		return succeed{}

	default:
		return report{}
	}
}

func deferredCall(val ssa.Value) (ssa.CallCommon, bool) {
	for _, ref := range *val.Referrers() {
		switch ref := ref.(type) {
		case ssa.Value:
			def, ok := deferredCall(ref)
			if ok {
				return def, true
			}
		case *ssa.Defer:
			return ref.Call, true
		}
	}

	return ssa.CallCommon{}, false
}

func resultantCall(val ssa.Value) *ssa.Call {
	call, ok := val.(*ssa.Call)
	if ok {
		return call
	}

	for _, ref := range *val.Referrers() {
		val, ok := ref.(ssa.Value)
		if !ok {
			continue
		}

		if c := resultantCall(val); c != nil {
			return c
		}
	}

	return nil
}

func (r runner) hasEmbeddedMockType(typ types.Type) bool {
	switch typ := typ.(type) {
	case *types.Pointer:
		return r.hasEmbeddedMockType(typ.Elem())
	case *types.Named:
		return r.hasEmbeddedMockType(typ.Underlying())
	case *types.Struct:
		for i := range typ.NumFields() {
			f := typ.Field(i)
			if !f.Embedded() {
				continue
			}

			named, ok := f.Type().(*types.Named)
			if !ok {
				continue
			}

			if r.isMockObj(named.Obj()) {
				return true
			}
		}

		return false
	default:
		return false
	}
}

func isTCleanupOrDefer(val ssa.Instruction) bool {
	switch val.(type) {
	case *ssa.Defer:
		return true
	case *ssa.Call:
	// Handled below.
	default:
		return false
	}

	call := val.(*ssa.Call)

	var name string
	var sig *types.Signature
	if call.Call.IsInvoke() {
		sig = call.Call.Method.Signature()
		name = call.Call.Method.Name()
	} else {
		sig = call.Call.Value.Type().(*types.Signature)
		name = call.Call.Value.Name()
	}

	if name != "Cleanup" || sig.Recv() == nil || sig.Params().Len() != 1 {
		return false
	}

	paramTyp, ok := sig.Params().At(0).Type().(*types.Signature)
	if !ok {
		return false
	}

	return paramTyp.Params().Len() == 0 && paramTyp.Results().Len() == 0
}

func (r runner) isAssertExpectations(call ssa.CallCommon) bool {
	if len(call.Args) < 1 {
		return false
	}

	if call.Value.Name() != "AssertExpectations" {
		return false
	}

	obj := typeutils.GetObjForPtrToNamedType(call.Args[0].Type())
	return r.isMockObj(obj)
}

func (r runner) isTest(call ssa.CallCommon) bool {
	if len(call.Args) < 1 {
		return false
	}

	if call.Value.Name() != "Test" {
		return false
	}

	obj := typeutils.GetObjForPtrToNamedType(call.Args[0].Type())
	return r.isMockObj(obj)
}
