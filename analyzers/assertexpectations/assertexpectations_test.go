package assertexpectations

import (
	"testing"

	"github.com/cszczepaniak/gomockcheck/names"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAssertExpectations_DefaultSetup(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(), "./defaulttype")
}

func TestAssertExpectations_CustomNames(t *testing.T) {
	analysistest.Run(
		t,
		analysistest.TestData(),
		New(names.QualifiedType{
			PkgPath: "example.com/customtype",
			Name:    "MyMockType",
		}),
		"./customtype",
	)
}
