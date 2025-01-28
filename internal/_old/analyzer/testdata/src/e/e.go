//nolint:all
package e

type External struct {
	A string
	B string
	c string
}

type ExternalExcluded struct {
	A string
	B string
	c string
}

func shouldPassAnonymousExcludedStruct() {
	_ = struct {
		A string
		B int
	}{}
}
