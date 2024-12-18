package names

import "go/types"

const (
	TestifyMockPkg = "github.com/stretchr/testify/mock"
	MockType       = "Mock"
)

func IsTestifyPkg(obj types.Object) bool {
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == TestifyMockPkg
}

func IsTestifySymbol(obj types.Object, name string) bool {
	return IsTestifyPkg(obj) && obj.Name() == name
}

func IsTestifyMock(obj types.Object) bool {
	return IsTestifySymbol(obj, MockType)
}

type QualifiedType struct {
	PkgPath string
	Name    string
}

func IsOneOf(obj types.Object, typs ...QualifiedType) bool {
	for _, t := range typs {
		if obj.Pkg().Path() == t.PkgPath && obj.Name() == t.Name {
			return true
		}
	}
	return false
}
