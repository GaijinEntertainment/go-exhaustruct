package main

import (
	"flag"

	"golang.org/x/tools/go/analysis/singlechecker"

	"dev.gaijin.team/go/exhaustruct/v5/analyzer"
)

func main() {
	flag.Bool("unsafeptr", false, "")

	singlechecker.Main(analyzer.NewAnalyzer())
}
