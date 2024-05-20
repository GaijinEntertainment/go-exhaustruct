package testdata

type TestType struct {
	Foo string
}

type TestTypeAlias = TestType

type TestAnonymousTypeAlias = struct {
	Foo string
}

func someFunc() any {
	type LocalType struct {
		Foo string
	}

	return LocalType{}
}

func someFunc2() any {
	a := make(map[string]struct{ a string })
	b := make(map[string]struct{})

	return a
}
