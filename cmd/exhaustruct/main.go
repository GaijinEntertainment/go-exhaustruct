package main

import (
	"flag"

	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/GaijinEntertainment/go-exhaustruct/pkg/analyzer"
)

func main() {
	flag.Bool("unsafeptr", false, "")
	singlechecker.Main(analyzer.Analyzer)
}
