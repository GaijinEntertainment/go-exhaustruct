package empty_patterns

type AllowedStruct struct {
	A string
	B int
}

type ForbiddenStruct struct {
	A string
	B int
}

type NestedAllowed struct {
	Inner AllowedStruct
	Value string
}

func shouldPassAllowedStructEmpty() {
	// Should pass because AllowedStruct matches the pattern
	_ = AllowedStruct{}
}

func shouldPassNestedAllowedEmpty() {
	// Should pass because NestedAllowed matches the pattern
	_ = NestedAllowed{}
}

func shouldPassPointerToAllowedEmpty() {
	// Should pass because AllowedStruct matches the pattern
	_ = &AllowedStruct{}
}

func shouldFailForbiddenStructEmpty() {
	// Should fail because ForbiddenStruct doesn't match the pattern
	_ = ForbiddenStruct{} // want "empty_patterns.ForbiddenStruct is missing fields A, B"
}

func shouldFailPointerToForbiddenEmpty() {
	// Should fail because ForbiddenStruct doesn't match the pattern
	_ = &ForbiddenStruct{} // want "empty_patterns.ForbiddenStruct is missing fields A, B"
}

func shouldPassAllowedInReturn() AllowedStruct {
	// Should pass because AllowedStruct matches the pattern
	return AllowedStruct{}
}

func shouldFailForbiddenInReturn() ForbiddenStruct {
	// Should fail because ForbiddenStruct doesn't match the pattern
	return ForbiddenStruct{} // want "empty_patterns.ForbiddenStruct is missing fields A, B"
}