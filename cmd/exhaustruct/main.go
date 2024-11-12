package main

import (
	"flag"

	"golang.org/x/tools/go/analysis/singlechecker"

	"dev.gaijin.team/go/go-exhaustruct/v4/analyzer"
)

func main() {
	flag.Bool("unsafeptr", false, "")

	a, err := analyzer.NewAnalyzer(nil, nil)
	if err != nil {
		panic(err)
	}

	singlechecker.Main(a)
}
