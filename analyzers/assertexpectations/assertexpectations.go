package assertexpectations

import (
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

func New() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "assertexpectations",
		Doc:      "Ensure that AssertExpectations is called on mock objects",
		Run:      run,
		Requires: []*analysis.Analyzer{buildssa.Analyzer},
	}
}

func run(pass *analysis.Pass) (any, error) {
	pssa := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	for _, f := range pssa.SrcFuncs {
		for _, b := range f.Blocks {
			for _, instr := range b.Instrs {
				alloc, ok := instr.(*ssa.Alloc)
				if !ok {
					continue
				}

				if !hasEmbeddedMockType(alloc.Type()) {
					continue
				}

				handleReferrers(pass, alloc)
			}
		}
	}

	return nil, nil
}

func handleReferrers(pass *analysis.Pass, alloc *ssa.Alloc) {
	for _, ref := range *alloc.Referrers() {
		continuation := handleReferrer(alloc, ref)
		switch continuation {
		case report:
			pass.Reportf(
				alloc.Pos(),
				"mocks must have an AssertExpectations registered in a defer or t.Cleanup",
			)
		case succeed:
			return
		case keepGoing:
		}
	}
}

type continuation int

const (
	keepGoing continuation = iota
	report
	succeed
)

func handleReferrer(alloc *ssa.Alloc, instr ssa.Instruction) continuation {
	switch ref := instr.(type) {
	case *ssa.Store:
		if ref.Addr != alloc {
			return keepGoing
		}

		_, ok := ref.Val.(*ssa.Alloc)
		if ok {
			// If the RHS of the store operation is an allocation, that means it's a struct literal
			// and we should analyze it.
			return keepGoing
		} else {
			// If it's not an allocation it could be something like a function call. If we're
			// storing the result of a function call, we'll analyze that function elsewhere so we
			// can assume it has set up AssertExpectations correctly.
			return succeed
		}
	case *ssa.MakeClosure:
		isCleanup := false
		for _, ref := range *ref.Referrers() {
			isCleanup = isTCleanup(ref)
			if isCleanup {
				break
			}
		}

		if !isCleanup {
			return report
		}

		var freeVar *ssa.FreeVar
		closure := ref.Fn.(*ssa.Function)
		for i, b := range ref.Bindings {
			if b == alloc {
				freeVar = closure.FreeVars[i]
				break
			}
		}

		foundAssertExpectations := false
		for _, ref := range *freeVar.Referrers() {
			val, ok := ref.(ssa.Value)
			if !ok {
				continue
			}

			c := resultantCall(val)
			if c != nil && isAssertExpectations(c.Call) {
				foundAssertExpectations = true
				break
			}
		}

		if !foundAssertExpectations {
			return report
		}
		return succeed

	case ssa.Value:
		deferredCall, ok := deferredCall(ref)
		if !ok || !isAssertExpectations(deferredCall) {
			return report
		}

		return succeed

	case *ssa.Defer:
		if isAssertExpectations(ref.Call) {
			return succeed
		}

		return report
	default:
		return report
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

func hasEmbeddedMockType(typ types.Type) bool {
	switch typ := typ.(type) {
	case *types.Pointer:
		return hasEmbeddedMockType(typ.Elem())
	case *types.Named:
		return hasEmbeddedMockType(typ.Underlying())
	case *types.Struct:
		for i := range typ.NumFields() {
			f := typ.Field(i)
			if !f.Embedded() {
				continue
			}

			if f.Name() != "Mock" {
				continue
			}

			named, ok := f.Type().(*types.Named)
			if !ok {
				continue
			}

			if named.Obj().Pkg().Path() == "github.com/stretchr/testify/mock" && named.Obj().Name() == "Mock" {
				return true
			}
		}

		return false
	default:
		return false
	}
}

func isTCleanup(val ssa.Instruction) bool {
	call, ok := val.(*ssa.Call)
	if !ok {
		return false
	}

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

func isAssertExpectations(call ssa.CallCommon) bool {
	if len(call.Args) < 1 {
		return false
	}

	if call.Value.Name() != "AssertExpectations" {
		return false
	}

	return isNamedPointer(call.Args[0].Type(), "github.com/stretchr/testify/mock", "Mock")
}

func isNamedPointer(typ types.Type, pkg, name string) bool {
	ptr, ok := typ.(*types.Pointer)
	if !ok {
		return false
	}

	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}

	return named.Obj().Pkg().Path() == pkg && named.Obj().Name() == name
}
