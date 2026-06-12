package main

import (
	"flag"

	"golang.org/x/tools/go/analysis/singlechecker"

	"dev.gaijin.team/go/exhaustruct/v5/analyzer"
)

func main() {
	// go vet passes -unsafeptr to every vettool; register a stub so flag
	// parsing does not fail when invoked via go vet -vettool.
	flag.Bool("unsafeptr", false, "")

	singlechecker.Main(analyzer.NewAnalyzer())
}
