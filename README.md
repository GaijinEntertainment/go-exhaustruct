<div align="center">

# exhaustruct

</div>

---

`exhaustruct` is a golang analyzer that finds structures with uninitialized fields

#### The "why?"

There is a similar linter [exhaustivestruct](https://github.com/mbilski/exhaustivestruct), but it is abandoned and not
optimal.

This linter can be called a successor of `exhaustivestruct`, and:

- it is at least **2.5+ times faster**, due to better algorithm;
- can receive `include` and/or `exclude` patterns;
- allows to mark fields as optional (not required to be filled on struct init), via field tag `exhaustruct:"optional"`;
- expects received patterns to be RegExp, therefore this package is not api-compatible with `exhaustivestruct`.

### Installation

```shell
go get -u github.com/GaijinEntertainment/go-exhaustruct/cmd/exhaustruct
```

### Usage

```
exhaustruct [-flag] [package]

Flags:
  -i value
        Regular expression to match struct packages and names, can receive multiple flags
  -e value
        Regular expression to exclude struct packages and names, can receive multiple flags
```

### Example

```go
// Package a.go
package a

type Shape struct {
	Length int
	Width  int

	volume int
	Perimeter int `exhaustruct:"optional"`
}

// valid
var a Shape = Shape{
	Length: 5,
	Width:  3,
	volume: 5,
}

// invalid, `volume` is missing
var b Shape = Shape{
	Length: 5,
	Width:  3,
}

// Package b.go
package b

import "a"

// valid
var b Shape = a.Shape{
	Length: 5,
	Width:  3,
}

// invalid, `Width` is missing
var b Shape = a.Shape{
	Length: 5,
}
```