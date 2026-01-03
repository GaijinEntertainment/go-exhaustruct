package analyzer

import (
	"flag"
	"strings"

	"dev.gaijin.team/go/golib/e"

	"dev.gaijin.team/go/exhaustruct/v4/internal/pattern"
)

type Config struct {
	// EnforceRx is a list of regular expressions to match type names that should be
	// checked. Anonymous structs can be matched by '<anonymous>' alias.
	//
	// Each regular expression must match the full type name, including package path.
	// For example, to match type `net/http.Cookie` regular expression should be
	// `.*/http\.Cookie`, but not `http\.Cookie`.
	EnforceRx       []string     `exhaustruct:"optional"`
	enforcePatterns pattern.List `exhaustruct:"optional"`

	// IgnoreRx is a list of regular expressions to match type names that should be
	// skipped from checking. Anonymous structs can be matched by '<anonymous>'
	// alias.
	//
	// Has precedence over EnforceRx.
	//
	// Each regular expression must match the full type name, including package path.
	// For example, to match type `net/http.Cookie` regular expression should be
	// `.*/http\.Cookie`, but not `http\.Cookie`.
	IgnoreRx       []string     `exhaustruct:"optional"`
	ignorePatterns pattern.List `exhaustruct:"optional"`

	// OptionalRx is a list of regular expressions to match type names where all
	// fields are treated as optional. Anonymous structs can be matched by '<anonymous>'
	// alias.
	//
	// Each regular expression must match the full type name, including package path.
	// For example, to match type `net/http.Cookie` regular expression should be
	// `.*/http\.Cookie`, but not `http\.Cookie`.
	OptionalRx       []string     `exhaustruct:"optional"`
	optionalPatterns pattern.List `exhaustruct:"optional"`

	// AllowEmpty allows empty structures, effectively excluding them from the check.
	AllowEmpty bool `exhaustruct:"optional"`

	// AllowEmptyRx is a list of regular expressions to match type names that should
	// be allowed to be empty. Anonymous structs can be matched by '<anonymous>'
	// alias.
	//
	// Each regular expression must match the full type name, including package path.
	// For example, to match type `net/http.Cookie` regular expression should be
	// `.*/http\.Cookie`, but not `http\.Cookie`.
	AllowEmptyRx       []string     `exhaustruct:"optional"`
	allowEmptyPatterns pattern.List `exhaustruct:"optional"`

	// AllowEmptyReturns allows empty structures in return statements.
	AllowEmptyReturns bool `exhaustruct:"optional"`

	// AllowEmptyDeclarations allows empty structures in variable declarations.
	AllowEmptyDeclarations bool `exhaustruct:"optional"`

	// ReportFullTypePath enables full package path in error messages instead of
	// short package name. This helps when configuring include/exclude patterns,
	// as import aliases can make short names ambiguous.
	ReportFullTypePath bool `exhaustruct:"optional"`

	// DebugCacheMetrics enables printing cache hit/miss metrics to stderr.
	DebugCacheMetrics bool `exhaustruct:"optional"`
}

// Prepare compiles all regular expression patterns into pattern lists for
// efficient matching.
func (c *Config) Prepare() error {
	var err error

	c.enforcePatterns, err = pattern.NewList(c.EnforceRx...)
	if err != nil {
		return e.NewFrom("compile enforce patterns", err)
	}

	c.ignorePatterns, err = pattern.NewList(c.IgnoreRx...)
	if err != nil {
		return e.NewFrom("compile ignore patterns", err)
	}

	c.optionalPatterns, err = pattern.NewList(c.OptionalRx...)
	if err != nil {
		return e.NewFrom("compile optional patterns", err)
	}

	c.allowEmptyPatterns, err = pattern.NewList(c.AllowEmptyRx...)
	if err != nil {
		return e.NewFrom("compile allow empty patterns", err)
	}

	return nil
}

// stringSliceFlag implements flag.Value interface for []string fields.
type stringSliceFlag struct {
	slice *[]string
}

func (s stringSliceFlag) String() string {
	if s.slice == nil {
		return ""
	}

	return strings.Join(*s.slice, ",")
}

func (s stringSliceFlag) Set(value string) error {
	*s.slice = append(*s.slice, value)
	return nil
}

// BindToFlagSet binds the config fields to the provided flag set.
func (c *Config) BindToFlagSet(fs *flag.FlagSet) *flag.FlagSet {
	fs.Var(stringSliceFlag{&c.EnforceRx}, "enforce-rx",
		"Regular expression to match type names that should be checked. "+
			"Anonymous structs can be matched by '<anonymous>' alias. "+
			"Each regex must match the full type name including package path. "+
			"Example: `.*/http\\.Cookie`. Can be used multiple times.")

	fs.Var(stringSliceFlag{&c.IgnoreRx}, "ignore-rx",
		"Regular expression to skip type names from checking, has precedence over -enforce-rx. "+
			"Anonymous structs can be matched by '<anonymous>' alias. "+
			"Each regex must match the full type name including package path. "+
			"Example: `.*/http\\.Cookie`. Can be used multiple times.")

	fs.Var(stringSliceFlag{&c.OptionalRx}, "optional-rx",
		"Regular expression to match type names where all fields are optional. "+
			"Anonymous structs can be matched by '<anonymous>' alias. "+
			"Each regex must match the full type name including package path. "+
			"Example: `.*/http\\.Cookie`. Can be used multiple times.")

	fs.BoolVar(&c.AllowEmpty, "allow-empty", c.AllowEmpty,
		"Allow empty structures, effectively excluding them from the check")

	fs.Var(stringSliceFlag{&c.AllowEmptyRx}, "allow-empty-rx",
		"Regular expression to match type names that should be allowed to be empty. "+
			"Anonymous structs can be matched by '<anonymous>' alias. "+
			"Each regex must match the full type name including package path. "+
			"Example: `.*/http\\.Cookie`. Can be used multiple times.")

	fs.BoolVar(&c.AllowEmptyReturns, "allow-empty-returns", c.AllowEmptyReturns,
		"Allow empty structures in return statements")

	fs.BoolVar(&c.AllowEmptyDeclarations, "allow-empty-declarations", c.AllowEmptyDeclarations,
		"Allow empty structures in variable declarations")

	fs.BoolVar(&c.ReportFullTypePath, "report-full-type-path", c.ReportFullTypePath,
		"Report full package path in error messages (e.g., 'net/http.Cookie' instead of 'http.Cookie'). "+
			"Useful for identifying types when configuring enforce/ignore patterns.")

	fs.BoolVar(&c.DebugCacheMetrics, "debug-cache-metrics", c.DebugCacheMetrics,
		"Print cache hit/miss metrics to stderr after each package analysis")

	return fs
}
