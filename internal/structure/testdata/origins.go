package testdata

// Original struct types.
type OriginalEmpty struct{}
type OriginalWithFields struct{ Field int }

// Aliases.
type AliasToOriginal = OriginalEmpty
type AliasToInt = int

// Derived types.
type DerivedFromOriginal OriginalEmpty
type DerivedFromAlias AliasToOriginal

// Generic types.
type GenericStruct[T any] struct{ Value T }
type GenericAlias[T any] = GenericStruct[T]

// Non-struct types (classified as derived in our taxonomy).
type MyInterface interface{ Method() }
type MyFunc func(int) string
type MySlice []int
type MyMap map[string]int
type MyChan chan int
