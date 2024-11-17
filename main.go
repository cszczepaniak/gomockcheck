package main

import (
	"github.com/cszczepaniak/gomockcheck/analyzers/assertexpectations"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(
		assertexpectations.New(),
	)
}
