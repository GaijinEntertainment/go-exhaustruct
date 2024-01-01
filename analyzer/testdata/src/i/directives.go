package i

func shouldNotFailOnIgnoreDirective() (Test, error) {
	//exhaustruct:ignore
	return Test{}, nil
}

func shouldNotFailOnIgnoreDirectivePlacedOnEOL() (Test, error) {
	return Test{}, nil //exhaustruct:ignore
}

func shouldNotPassExcludedButEnforced() {
	//exhaustruct:enforce
	_ = TestExcluded{} // want "i.TestExcluded is missing fields A, B"
}
