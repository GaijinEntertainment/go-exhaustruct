package i

import (
	"e"
)

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

	// directive in a slice with type
	_ = []any{
		Test{}, // want "i.Test is missing fields A, B, C, D"
		Test{}, //exhaustruct:ignore
		Test{}, // want "i.Test is missing fields A, B, C, D"
	}
	// directive in a slice without type
	_ = []Test{
		{}, // want "i.Test is missing fields A, B, C, D"
		{}, //exhaustruct:ignore
		{}, // want "i.Test is missing fields A, B, C, D"
	}
	// directive in a map
	_ = map[string]any{
		"a": Test{}, // want "i.Test is missing fields A, B, C, D"
		"b": Test{}, //exhaustruct:ignore
		"c": Test{}, // want "i.Test is missing fields A, B, C, D"
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
}

func shouldFailOnMisappliedDirectives() {
	// wrong directive name
	//exhaustive:enforce
	_ = TestExcluded{B: 0}
}

func shouldHandleDirectivesOnEmbedded() {
	_ = Test2{
		//exhaustruct:ignore
		External: e.External{},
		//exhaustruct:enforce
		Embedded: Embedded{}, // want "i.Embedded is missing fields E, F, g, H"
	}

	_ = Test2{
		//exhaustruct:ignore
		External: e.External{},
		//exhaustruct:ignore
		Embedded: Embedded{},
	}
}
