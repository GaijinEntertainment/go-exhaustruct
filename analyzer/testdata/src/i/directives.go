package i

func excludedConsumer(e TestExcluded) string {
	return e.A
}

type TestIncludedEmbedded struct {
	A string
	Embedded
}

type TestExcludedEmbedded struct {
	A string
	Embedded
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

	// directive on embedded struct
	_ = TestIncludedEmbedded{
		A: "",
		//exhaustruct:ignore
		Embedded: Embedded{},
	}

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

	// directive on embedded and parent struct
	//exhaustruct:enforce
	_ = TestExcludedEmbedded{ // want "i.TestExcludedEmbedded is missing field A"
		//exhaustruct:enforce
		Embedded: Embedded{}, // want "i.Embedded is missing fields E, F, g, H"
	}

	// directive on embedded struct
	_ = TestExcludedEmbedded{
		A: "",
		//exhaustruct:enforce
		Embedded: Embedded{}, // want "i.Embedded is missing fields E, F, g, H"
	}

}

func shouldFailOnMisappliedDirectives() {
	// wrong directive name
	//exhaustive:enforce
	_ = TestExcluded{B: 0}
}
