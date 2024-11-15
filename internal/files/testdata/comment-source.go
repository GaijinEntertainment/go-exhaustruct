package testdata

// Test before structure name.
type Test struct { // after structure name
	Foo string // after field declaration
	// before field declaration
	// miltiline comment
	//
	// with empty lines
	Bar string // after field declaration [2]
	Baz string // after field declaration [3]
}
