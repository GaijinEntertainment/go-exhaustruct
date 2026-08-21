// Package user consumes a dependency whose directives are malformed. The
// malformed directives belong to the dependency, so nothing about them is
// reported here.
package user

import "testdata/config/dep_directives/dep"

func shouldReportOnlyItsOwnFindings() {
	_ = dep.Shared{A: 1} // want "dep.Shared is missing field B"
}
