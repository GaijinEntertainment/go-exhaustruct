package directives

import (
	"i"
)

func excludedConsumer(e i.TestExcluded) string {
	return e.A
}

func shouldFailEnforcementDirective() {
	//exhaustruct:enforce
	_ = i.TestExcluded{ // want "i.TestExcluded is missing field A"
		B: 0,
	}

	_ = i.TestExcluded{ // want "i.TestExcluded is missing field A"
		B: 0,
	} //exhaustruct:enforce

	_ = excludedConsumer(
		//exhaustruct:enforce
		i.TestExcluded{ // want "i.TestExcluded is missing field A"
			B: 0,
		},
	)

	_ =
		//exhaustruct:enforce
		i.Test{ // want "i.Test is missing field A"
			B: 0,
			C: 0.0,
			D: false,
			E: "",
		}
}

func shouldSucceedIgnoreDirective() {
	//exhaustruct:ignore
	_ = i.Test3{
		B: 0,
	}

	_ =
		//exhaustruct:ignore
		i.Test3{
			B: 0,
		}

	//exhaustruct:ignore
	_ = i.Test2{
		//exhaustruct:ignore
		Embedded: i.Embedded{},
	}
}

func misappliedDirectives() {
	// associated with wrong parent node
	//exhaustruct:enforce
	_ = excludedConsumer(i.TestExcluded{
		B: 0,
	})

	// wrong directive name
	//exhaustive:enforce
	_ = i.TestExcluded{
		B: 0,
	}
}
