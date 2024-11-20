package mocksetup

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestMockSetup(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), New(), "./...")
}

func TestMockSetup_SuggestedFixes(t *testing.T) {
	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), New(), "./suggestedfixes")
}
