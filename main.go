package main

import (
	"github.com/cszczepaniak/gomockcheck/analyzers/assertexpectations"
	"github.com/cszczepaniak/gomockcheck/analyzers/mocksetup"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(
		assertexpectations.New(),
		mocksetup.New(),
	)
}
