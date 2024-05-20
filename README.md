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
        Regular expression to match type names, can receive multiple flags.
        Anonymous structs can be matched by '<anonymous>' alias.
        4ex:
                github.com/GaijinEntertainment/go-exhaustruct/v3/analyzer\.<anonymous>
                github.com/GaijinEntertainment/go-exhaustruct/v3/analyzer\.TypeInfo
        
  -e value
        Regular expression to exclude type names, can receive multiple flags.
        Anonymous structs can be matched by '<anonymous>' alias.
        4ex:
                github.com/GaijinEntertainment/go-exhaustruct/v3/analyzer\.<anonymous>
                github.com/GaijinEntertainment/go-exhaustruct/v3/analyzer\.TypeInfo
```

#### Comment directives

`exhaustruct` supports comment directives to mark individual structures as ignored during linting or enforce it's check
regardless global configuration. Comment directives have precedence over global configuration.

- **`//exhaustruct:ignore`** - ignore structure during linting
- **`//exhaustruct:enforce`** - enforce structure check during linting, even in case global configuration says it should
  be ignored.

> Note: all directives can be placed in the end of structure declaration or on the line above it.
>
> Also, any additional comment can be placed same line right after the directive or anywhere around it, but directive
> should be at the very beginning of the line. It is _recommended_ to comment directives, especially when ignoring
> structures - it will help to understand the reason later.

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

### Errors handling

In order to avoid unnecessary noise, when dealing with non-pointer types returned along with errors - `exhaustruct` will
ignore non-error types, checking only structures satisfying `error` interface.

```go
package main

import "errors"

type Shape struct {
	Length int
	Width  int
}

func NewShape() (Shape, error) {
	return Shape{}, errors.New("error") // will not raise an error
}

type MyError struct {
	Err error
}

func (e MyError) Error() string {
    return e.Err.Error()
}

func NewSquare() (Shape, error) {
    return Shape{}, MyError{Err: errors.New("error")} // will not raise an error
}

func NewCircle() (Shape, error) {
    return Shape{}, MyError{} // will raise "main.MyError is missing field Err"
}

```
