package i

func excludedConsumer(e TestExcluded) string {
	return e.A
}

func shouldNotFailOnIgnoreDirective() (Test, error) {
	// directive on previous line
	//exhaustruct:ignore
	_ = Test2{}

	// directive at the end of the line
	_ = Test{} //exhaustruct:ignore

	// some style weirdness
	_ =
		//exhaustruct:ignore
		Test3{
			B: 0,
		}

	// directive after the literal
	_ = Test{
		B: 0,
	} //exhaustruct:ignore

	//exhaustruct:ignore
	return Test{}, nil
}

func shouldFailOnExcludedButEnforced() {
	// directive on previous line associated with different ast leaf
	//exhaustruct:enforce
	_ = excludedConsumer(TestExcluded{B: 0}) // want "i.TestExcluded is missing field A"

	// initially excluded, but enforced
	//exhaustruct:enforce
	_ = TestExcluded{} // want "i.TestExcluded is missing fields A, B"
}

func shouldFailOnMisappliedDirectives() {
	// wrong directive name
	//exhaustive:enforce
	_ = TestExcluded{B: 0}
}
