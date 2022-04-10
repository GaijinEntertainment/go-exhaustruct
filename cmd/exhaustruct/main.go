package main

import (
	"flag"

	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/GaijinEntertainment/go-exhaustruct/v2/pkg/analyzer"
)

func main() {
	flag.Bool("unsafeptr", false, "")

	a, err := analyzer.NewAnalyzer([]string{}, []string{})
	if err != nil {
		panic(err)
	}

	singlechecker.Main(a)
}
