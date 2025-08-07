package main

import (
	"flag"

	"golang.org/x/tools/go/analysis/singlechecker"

	"dev.gaijin.team/go/exhaustruct/v4/analyzer"
)

func main() {
	flag.Bool("unsafeptr", false, "")

	a, err := analyzer.NewAnalyzer(analyzer.Config{})
	if err != nil {
		panic(err)
	}

	singlechecker.Main(a)
}
