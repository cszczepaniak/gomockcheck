package assertexpectations

import (
	"fmt"
	"go/token"
	"go/types"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

type MockType struct {
	Pkg  string
	Name string
}

func New(typs ...MockType) *analysis.Analyzer {
	r := runner{
		types: slices.Concat([]MockType{{
			Pkg:  "github.com/stretchr/testify/mock",
			Name: "Mock",
		}}, typs),
	}

	return &analysis.Analyzer{
		Name:     "assertexpectations",
		Doc:      "Ensure that AssertExpectations is called on mock objects",
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
	types []MockType
}

func (r runner) isMockObjName(obj types.Object) bool {
	for _, typ := range r.types {
		if obj.Name() == typ.Name {
			return true
		}
	}

	return false
}

func (r runner) isMockObj(obj types.Object) bool {
	for _, typ := range r.types {
		if obj.Pkg().Path() == typ.Pkg && obj.Name() == typ.Name {
			return true
		}
	}

	return false
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

				for _, instr := range *alloc.Referrers() {
					val, ok := instr.(ssa.Value)
					if ok {
						debugf("%s: %T %v\n", val.Name(), instr, instr)
					} else {
						debugf("%T %v\n", instr, instr)
					}
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
			// and we should analyze it.
			return keepGoing{pos: ref.Val.Pos()}
		} else {
			// If it's not an allocation it could be something like a function call. If we're
			// storing the result of a function call, we'll analyze that function elsewhere so we
			// can assume it has set up AssertExpectations correctly.
			return succeed{}
		}
	case *ssa.MakeClosure:
		// This is the case that we're referring to the mock in a closure. We'll check to see if this is
		// a closure passed into a t.Cleanup, and if that closure calls AssertExpectations.
		isCleanup := false
		for _, ref := range *ref.Referrers() {
			isCleanup = isTCleanup(ref)
			if isCleanup {
				break
			}
		}

		if !isCleanup {
			return report{}
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
			if c != nil && r.isAssertExpectations(c.Call) {
				foundAssertExpectations = true
				break
			}
		}

		if !foundAssertExpectations {
			return report{}
		}
		return succeed{}

	case ssa.Value:
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

			if !r.isMockObjName(f) {
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

func (r runner) isAssertExpectations(call ssa.CallCommon) bool {
	if len(call.Args) < 1 {
		return false
	}

	if call.Value.Name() != "AssertExpectations" {
		return false
	}

	ptr, ok := call.Args[0].Type().(*types.Pointer)
	if !ok {
		return false
	}

	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}

	return r.isMockObj(named.Obj())
}
