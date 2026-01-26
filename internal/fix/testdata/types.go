package testdata

type Nested struct {
	Value   int
	Name    string
	Enabled bool
}

type DeepNested struct {
	Inner   Nested
	Label   string
	Count   int
	Tags    []string
}

type TestStruct struct {
	stringField    string
	intField       int
	boolField      bool
	float64Field   float64
	pointerField   *int
	sliceField     []string
	mapField       map[string]int
	chanField      chan int
	interfaceField interface{}
	funcField      func()
	structField    Nested
}

type NestedStruct struct {
	Name   string
	Nested Nested
	Deep   DeepNested
	Ptr    *Nested
	Slice  []Nested
	Map    map[string]Nested
}
