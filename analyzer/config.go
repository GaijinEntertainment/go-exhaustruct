package analyzer

import (
	"flag"

	"dev.gaijin.team/go/exhaustruct/v4/internal/pattern"
)

type Config struct {
	// EnforcePatterns is a list of regular expressions to match type names that
	// should be checked. Anonymous structs can be matched by '<anonymous>' alias.
	//
	// Each regular expression must match the full type name, including package path.
	// For example, to match type `net/http.Cookie` regular expression should be
	// `.*/http\.Cookie`, but not `http\.Cookie`.
	EnforcePatterns pattern.List `exhaustruct:"optional"`

	// IgnorePatterns is a list of regular expressions to match type names that
	// should be skipped from checking. Anonymous structs can be matched by
	// '<anonymous>' alias.
	//
	// Has precedence over EnforcePatterns.
	//
	// Each regular expression must match the full type name, including package path.
	// For example, to match type `net/http.Cookie` regular expression should be
	// `.*/http\.Cookie`, but not `http\.Cookie`.
	IgnorePatterns pattern.List `exhaustruct:"optional"`

	// OptionalPatterns is a list of regular expressions to match type names where
	// all fields are treated as optional. Anonymous structs can be matched by
	// '<anonymous>' alias.
	//
	// Each regular expression must match the full type name, including package path.
	// For example, to match type `net/http.Cookie` regular expression should be
	// `.*/http\.Cookie`, but not `http\.Cookie`.
	OptionalPatterns pattern.List `exhaustruct:"optional"`

	// AllowEmpty allows empty structures, effectively excluding them from the check.
	AllowEmpty bool `exhaustruct:"optional"`

	// AllowEmptyPatterns is a list of regular expressions to match type names that
	// should be allowed to be empty. Anonymous structs can be matched by
	// '<anonymous>' alias.
	//
	// Each regular expression must match the full type name, including package path.
	// For example, to match type `net/http.Cookie` regular expression should be
	// `.*/http\.Cookie`, but not `http\.Cookie`.
	AllowEmptyPatterns pattern.List `exhaustruct:"optional"`

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

	// ExplicitMode enables opt-in checking. When true, only types marked with
	// //exhaustruct:enforce directive or matching enforce-rx patterns are checked.
	ExplicitMode bool `exhaustruct:"optional"`
}

// BindToFlagSet binds the config fields to the provided flag set.
func (c *Config) BindToFlagSet(fs *flag.FlagSet) *flag.FlagSet {
	fs.Var(&c.EnforcePatterns, "enforce-rx",
		"Regular expression to match type names that should be checked. "+
			"Anonymous structs can be matched by '<anonymous>' alias. "+
			"Each regex must match the full type name including package path. "+
			"Example: `.*/http\\.Cookie`. Can be used multiple times.")

	fs.Var(&c.IgnorePatterns, "ignore-rx",
		"Regular expression to skip type names from checking, has precedence over -enforce-rx. "+
			"Anonymous structs can be matched by '<anonymous>' alias. "+
			"Each regex must match the full type name including package path. "+
			"Example: `.*/http\\.Cookie`. Can be used multiple times.")

	fs.Var(&c.OptionalPatterns, "optional-rx",
		"Regular expression to match type names where all fields are optional. "+
			"Anonymous structs can be matched by '<anonymous>' alias. "+
			"Each regex must match the full type name including package path. "+
			"Example: `.*/http\\.Cookie`. Can be used multiple times.")

	fs.BoolVar(&c.AllowEmpty, "allow-empty", c.AllowEmpty,
		"Allow empty structures, effectively excluding them from the check")

	fs.Var(&c.AllowEmptyPatterns, "allow-empty-rx",
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

	fs.BoolVar(&c.ExplicitMode, "explicit", c.ExplicitMode,
		"Enable explicit mode: only check types marked with //exhaustruct:enforce "+
			"directive or matching -enforce-rx patterns")

	return fs
}
