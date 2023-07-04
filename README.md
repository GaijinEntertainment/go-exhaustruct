<div align="center">

# exhaustruct

![Package Version](https://img.shields.io/github/v/release/GaijinEntertainment/go-exhaustruct?style=flat-square)
![Go version](https://img.shields.io/github/go-mod/go-version/GaijinEntertainment/go-exhaustruct?style=flat-square)
![GitHub Workflow Status (with branch)](https://img.shields.io/github/actions/workflow/status/GaijinEntertainment/go-exhaustruct/ci.yml?branch=master)
![License](https://img.shields.io/github/license/GaijinEntertainment/go-exhaustruct?style=flat-square)


</div>

---

`exhaustruct` is a golang analyzer that finds structures with uninitialized fields

### Installation

```shell
go get -u github.com/GaijinEntertainment/go-exhaustruct/v3/cmd/exhaustruct
```

### Usage

```
exhaustruct [-flag] [package]

Flags:
  -i value
        Regular expression to match struct definitions, can receive multiple flags.

		Default: empty (match all struct definitions)
  -e value
        Regular expression to exclude struct definitions, can receive multiple flags

		Default: empty (exclude no struct definitions)
  -filter-anon bool
        Only enforce exhaustive anonymous struct literals when the package and struct pseudonym
		of the literal are matched by the inclusion and exclusion filters; otherwise all anonymous
		struct literals will be subject to enforcement. Note that anonymous structs are identified
		by the pseudonym "<anonymous>" for filtering purposes, e.g. "github.com/my/pkg.<anonymous>"

		Default: false (match all anonymous struct literals)
```

### Example

```go
// Package a.go
package a

type Shape struct {
	Length int
	Width  int

	volume    int
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
