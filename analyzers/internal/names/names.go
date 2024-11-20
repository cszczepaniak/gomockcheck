package names

import "go/types"

const (
	TestifyMockPkg = "github.com/stretchr/testify/mock"
	MockType       = "Mock"
)

func IsTestifyPkg(obj types.Object) bool {
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == TestifyMockPkg
}

func IsTestifyMock(obj types.Object) bool {
	return IsTestifyPkg(obj) && obj.Name() == MockType
}
