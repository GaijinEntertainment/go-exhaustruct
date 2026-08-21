// Package ignored_tags covers deprecated tags on a type the configuration
// excludes. Migrating them is a one-time move off v4 syntax, and a tag left
// behind is a tag the next reader has to rule on, so what the configuration
// checks does not decide what the migration reaches.
package ignored_tags

type ExcludedByPattern struct {
	// want +1 `struct tag "exhaustruct" is not supported anymore`
	Field string `exhaustruct:"optional"`
}

type Reported struct {
	// want +1 `struct tag "exhaustruct" is not supported anymore`
	Field string `exhaustruct:"optional"`
}

// A literal enforcing the type checks it despite the pattern, so the field its
// dead tag was written to spare is reported as missing.
type ExcludedButEnforcedAtUse struct {
	// want +1 `struct tag "exhaustruct" is not supported anymore`
	Field string `exhaustruct:"optional"`
	Other string
}

func enforcedAtUse() {
	//exhaustruct:enforce
	_ = ExcludedButEnforcedAtUse{} // want `ExcludedButEnforcedAtUse is missing fields Field, Other`
}

// An anonymous struct the configuration ignores carries its tags into the
// migration all the same.
func ignoredAnonymous() {
	_ = struct {
		// want +1 `struct tag "exhaustruct" is not supported anymore`
		Field string `exhaustruct:"optional"`
	}{Field: "x"}
}
