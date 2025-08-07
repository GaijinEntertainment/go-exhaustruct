<div align="center">

# exhaustruct

![Package Version](https://img.shields.io/github/v/release/GaijinEntertainment/go-exhaustruct?style=flat-square)
![Go version](https://img.shields.io/github/go-mod/go-version/GaijinEntertainment/go-exhaustruct?style=flat-square)
![GitHub Workflow Status (with branch)](https://img.shields.io/github/actions/workflow/status/GaijinEntertainment/go-exhaustruct/ci.yml?branch=master)
![License](https://img.shields.io/github/license/GaijinEntertainment/go-exhaustruct?style=flat-square)

</div>

---

`exhaustruct` is a golang analyzer that finds structures with uninitialized fields

## RENAME WARNING

Package being renamed to `dev.gaijin.team/go/exhaustruct/v4` and all further updates will be published under this name.

### Installation

```shell
go get -u dev.gaijin.team/go/exhaustruct/v4/cmd/exhaustruct
```

### Usage

```
exhaustruct [-flag] [package]

Flags:
  -i pattern | -include-rx pattern
        Regular expression to match type names that should be processed.
        Anonymous structs can be matched by '<anonymous>' alias.
        Example: .*/http\.Cookie

  -e pattern | -exclude-rx pattern
        Regular expression to exclude type names from processing, has precedence over -include.
        Anonymous structs can be matched by '<anonymous>' alias.
        Example: .*/http\.Cookie

  -allow-empty
        Allow empty structures globally, effectively excluding them from the check

  -allow-empty-returns
        Allow empty structures in return statements

  -allow-empty-declarations
        Allow empty structures in variable declarations

  -allow-empty-rx pattern
        Regular expression to match type names that should be allowed to be empty.
        Anonymous structs can be matched by '<anonymous>' alias.
        Example: .*/http\.Cookie
```

If you're using [golangci-lint](https://golangci-lint.run/), refer to
the [linters settings](https://golangci-lint.run/usage/linters/#exhaustruct) for the most up-to-date configuration
guidance.

#### Comment directives

`exhaustruct` supports comment directives to mark individual structure declarations as ignored during linting or enforce
it's check regardless global configuration. Comment directives have precedence over global configuration.

- **`//exhaustruct:ignore`** - ignore structure during linting
- **`//exhaustruct:enforce`** - enforce structure check during linting, even in case global configuration says it should
  be ignored.

> Note: all directives can be placed on the line above opening bracket or on the same line.
>
> Also, any additional comment can be placed same line right after the directive or anywhere around it, but directive
> should be at the very beginning of the line. It is _recommended_ to comment directives, especially when ignoring
> structures - it will help to understand the reason later.

### Examples

#### Basic Usage

By default, linter will check all structures and report if any accessible field initialization is missing.

```go
package main

type Config struct {
	Host     string
	Port     int
	Database string
}

// Without any flags - requires all fields
func createConfig() Config {
	return Config{} // ERROR: missing fields Host, Port, Database
}

func createValidConfig() Config {
	return Config{
		Host:     "localhost",
		Port:     5432,
		Database: "mydb",
	}
}

```

#### Empty Allowance Options

##### 1. Global Empty Allowance (`-allow-empty`)

**Rationale**: Useful when working with configuration structs that have sensible defaults, or when migrating codebases
where empty structs are acceptable everywhere.

```bash
exhaustruct -allow-empty ./...
```

```go
package main

// With -allow-empty: ALL empty structs are allowed
func createConfig() Config {
	return Config{} // OK: empty structs allowed globally
}

var globalConfig = Config{} // OK: empty structs allowed globally

func processConfigs() []Config {
	return []Config{{}} // OK: empty structs allowed globally
}

```

##### 2. Return Statement Allowance (`-allow-empty-returns`)

**Rationale**: Common pattern where functions return zero-value structs in error conditions, while still enforcing
proper initialization in other contexts.  
This option allows zero-value structs only as a **direct** child of return statement.

```bash
exhaustruct -allow-empty-returns ./...
```

```go
package main

// With -allow-empty-returns: empty structs allowed only in return statements
func createConfig() Config {
	return Config{} // OK: empty struct in return statement
}

func initializeConfig() {
	var config = Config{} // ERROR: empty struct in variable declaration
	_ = config
}

func processConfigs() []Config {
	return []Config{{}} // ERROR: empty struct in slice literal (not direct child of return statement)
}

```

##### 3. Variable Declaration Allowance (`-allow-empty-declarations`)

**Rationale**: Allows empty initialization in variable declarations while enforcing proper initialization in other
contexts. Useful for gradual struct population patterns.
This option allows zero-value structs only as a **direct** child of variable declaration statement.

```bash
exhaustruct -allow-empty-declarations ./...
```

```go
package main

// With -allow-empty-declarations: empty structs allowed in variable declarations
func initializeConfig() {
	var config = Config{} // OK: empty struct in variable declaration
	config := Config{}    // OK: empty struct in short variable declaration
	ptr := &Config{}      // OK: empty struct in pointer declaration
	_ = config
	_ = ptr
}

func createConfig() Config {
	return Config{} // ERROR: empty struct in return statement
}

func processConfigs() []Config {
	return []Config{{}} // ERROR: empty struct in slice literal
}
```

##### 4. Pattern-Based Allowance (`-allow-empty-include`)

**Rationale**: Granular control allowing empty structs only for specific types, typically third-party libraries or
specific patterns where empty initialization is common practice.

```bash
exhaustruct -allow-empty-include ".*Config.*" -allow-empty-include ".*Options.*" ./...
```

```go
package main

type DatabaseConfig struct {
	Host string
	Port int
}

type ServerOptions struct {
	Timeout  int
	MaxConns int
}

type UserData struct {
	Name  string
	Email string
}

func example() {
	// OK: matches .*Config.* pattern
	config := DatabaseConfig{}

	// OK: matches .*Options.* pattern
	opts := ServerOptions{}

	// ERROR: doesn't match any pattern
	user := UserData{}

	_ = config
	_ = opts
	_ = user
}

```

#### Errors handling

In order to avoid unnecessary noise, when dealing with non-pointer types returned along with errors - `exhaustruct` will
ignore non-error types, in case return statement contains non-nil value that satisfies `error` interface.

```go
package main

import (
	"errors"
)

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
	return Shape{}, &MyError{Err: errors.New("error")} // will not raise an error
}

func NewCircle() (Shape, error) {
	return Shape{}, &MyError{} // will raise "main.MyError is missing field Err"
}

```
