package main

import (
	"flag"

	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/GaijinEntertainment/go-exhaustruct/v3/analyzer"
)

func main() {
	flag.Bool("unsafeptr", false, "")

	a, err := analyzer.NewAnalyzer(nil, nil, true)
	if err != nil {
		panic(err)
	}

	singlechecker.Main(a)
}
