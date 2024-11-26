package allow_empty

type Test struct {
	A string
	B string
}

func shouldPassEmptyStruct() Test {
	return Test{}
}

func shouldFailGeneric() {
	_ = Test{ // want "allow_empty.Test is missing field B"
		A: "a",
	}
}

func shouldPassGeneric() {
	_ = Test{
		A: "a",
		B: "b",
	}
}
