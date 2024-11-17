package assertexpectations

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAssertExpectations(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(), "./...")
}
