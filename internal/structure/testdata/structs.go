package testdata

import (
	"net/http"
)

type testStruct struct {
	// some random comment

	ExportedRequired   int
	unexportedRequired int

	ExportedOptional   int `exhaustruct:"optional"`
	unexportedOptional int `exhaustruct:"optional"`
}

var (
	_unnamed = testStruct{1, 2, 3, 4}
	_named   = testStruct{
		ExportedRequired:   1,
		unexportedRequired: 2,
		ExportedOptional:   3,
		unexportedOptional: 4,
	}
	_unnamedIncomplete = testStruct{1}
	_namedIncomplete1  = testStruct{
		ExportedRequired: 1,
		ExportedOptional: 3,
	}
	_namedIncomplete2 = testStruct{
		ExportedOptional:   3,
		unexportedOptional: 4,
	}
)

//exhaustruct:ignore
type NamedTestType struct {
	Required string
	Optional string `exhaustruct:"optional"`
	//exhaustruct:optional
	CommentOptional string
}

//exhaustruct:enforce
type NamedTestType2 NamedTestType

//exhaustruct:ignore
type NamedTestType3 NamedTestType2

//exhaustruct:enforce
type AliasTestType = NamedTestType

//exhaustruct:ignore
type AliasImportedTestType = http.Transport

//exhaustruct:ignore
type AnonymousAliasTestType = struct {
	Required string
}

//exhaustruct:enforce
type AnonymousAliasEmptyTestType = struct{}

//exhaustruct:ignore
var (
	_NamedTestTypeVariable  = NamedTestType{}
	_NamedTestType2Variable = NamedTestType2{}
	_NamedTestType3Variable = NamedTestType3{}

	_AliasTestTypeVariable         = AliasTestType{}
	_AliasImportedTestTypeVariable = AliasImportedTestType{}

	_AnonymousAliasTestTypeVariable      = AnonymousAliasTestType{}
	_AnonymousAliasEmptyTestTypeVariable = AnonymousAliasEmptyTestType{}

	//exhaustruct:ignore i'm searching this one
	_AnonymousTestTypeVariable = struct {
		Required string
		//exhaustruct:optional
		CommentOptional string
	}{} //exhaustruct:ignore i'm searching that one

	//exhaustruct:enforce
	_AnonymousTestTypeVariable2 = struct{}{}
)

func someFunc() (any, any) {
	type LocalType struct {
		//exhaustruct:ignore
		Foo string
		//exhaustruct:enforce
		Bar string
	}

	_LocalTypeVariable := LocalType{}

	_AnonymousTypeVariable := struct{}{}

	return _LocalTypeVariable, _AnonymousTypeVariable
}
