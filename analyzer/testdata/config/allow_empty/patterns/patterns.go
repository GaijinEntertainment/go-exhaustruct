// Package patterns tests AllowEmptyRx pattern behavior.
// Only types matching the pattern allow empty struct literals.
package patterns

// AllowedStruct matches the allow-empty pattern.
type AllowedStruct struct {
	A string
	B int
}

// ForbiddenStruct does not match the pattern.
type ForbiddenStruct struct {
	A string
	B int
}

// NestedAllowed contains AllowedStruct.
type NestedAllowed struct {
	Inner AllowedStruct
	Value string
}

func shouldPassAllowedEmpty() {
	_ = AllowedStruct{}
	_ = &AllowedStruct{}
}

func shouldPassNestedAllowedEmpty() {
	_ = NestedAllowed{}
}

func shouldPassAllowedInReturn() AllowedStruct {
	return AllowedStruct{}
}

func shouldFailForbiddenEmpty() {
	_ = ForbiddenStruct{}  // want "patterns.ForbiddenStruct is missing fields A, B"
	_ = &ForbiddenStruct{} // want "patterns.ForbiddenStruct is missing fields A, B"
}

func shouldFailForbiddenInReturn() ForbiddenStruct {
	return ForbiddenStruct{} // want "patterns.ForbiddenStruct is missing fields A, B"
}
