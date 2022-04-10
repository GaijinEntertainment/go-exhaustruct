package main

import (
	"flag"

	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/GaijinEntertainment/go-exhaustruct/v2/pkg/analyzer"
)

func main() {
	flag.Bool("unsafeptr", false, "")

	singlechecker.Main(analyzer.MustNewAnalyzer([]string{}, []string{}))
}
