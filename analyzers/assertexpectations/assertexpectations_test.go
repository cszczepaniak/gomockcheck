package assertexpectations

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAssertExpectations_DefaultSetup(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(), "./defaulttype")
}

func TestAssertExpectations_CustomNames(t *testing.T) {
	analysistest.Run(
		t,
		analysistest.TestData(),
		New(MockType{
			Pkg:  "example.com/customtype",
			Name: "MyMockType",
		}),
		"./customtype",
	)
}
