package analyzer

import (
	"golang.org/x/tools/go/analysis"
)

type analyzer struct {
}

type Config struct {
	// Include is a list of regular expressions that match the names of the
	// structures that should be checked. Anonymous structs can be matched by
	// '<anonymous>' alias.
	//
	// Include list has precedence over the Exclude list.
	Include []string

	// Exclude is a list of regular expressions that match the names of the
	// structures that should be excluded from the check. Anonymous structs can be
	// matched by '<anonymous>' alias.
	//
	// Include list has precedence over the Exclude list.
	Exclude []string

	// Explicit is a flag indication that all fields of all structures are optional
	// by default. In this mode developer will have to explicitly put `<TODO: FILL ME LATER>`
	// comment directive to make check structure or field.
	Explicit bool
}

func NewAnalyzer(cfg Config) (*analysis.Analyzer, error) {
	a := analyzer{}

	//exhaustruct:exclude
	return &analysis.Analyzer{
		Name: "exhaustruct",
		Doc:  "Checks if all structure fields are initialized",
		Run:  a.run,
	}, nil
}

func (a *analyzer) run(pass *analysis.Pass) (interface{}, error) {
	return nil, nil
}
