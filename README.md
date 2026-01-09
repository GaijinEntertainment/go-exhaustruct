<div align="center">

# exhaustruct

![Package Version](https://img.shields.io/github/v/release/GaijinEntertainment/go-exhaustruct?style=flat-square)
![Go version](https://img.shields.io/github/go-mod/go-version/GaijinEntertainment/go-exhaustruct?style=flat-square)
![GitHub Workflow Status (with branch)](https://img.shields.io/github/actions/workflow/status/GaijinEntertainment/go-exhaustruct/ci.yml?branch=master)
![License](https://img.shields.io/github/license/GaijinEntertainment/go-exhaustruct?style=flat-square)

</div>

---

`exhaustruct` is a golang analyzer that finds structures with uninitialized fields.

If you're using [golangci-lint](https://golangci-lint.run/), refer to
the [linters settings](https://golangci-lint.run/usage/linters/#exhaustruct)
for the most up-to-date configuration guidance.

## Installation

```shell
go install dev.gaijin.team/go/exhaustruct/v4/cmd/exhaustruct@latest
```

## How It Works

The analyzer inspects struct literals in your code and reports when required
fields are not initialized:

```go
type User struct {
    Name  string
    Email string
    Age   int
}

func example() {
    _ = User{Name: "alice"} // ERROR: missing fields Email, Age
    _ = User{Name: "alice", Email: "alice@example.com", Age: 30} // OK
}
```

## Modes of Operation

### Implicit Mode (Default)

By default, **all** struct literals are checked. This ensures complete
initialization across your codebase. Use ignore patterns or directives to
exclude specific types or literals.

### Explicit Mode

With `-explicit` flag, the analyzer only checks structs that are explicitly
marked for enforcement — either via `//exhaustruct:enforce` directive or
`-enforce-rx` patterns. This is useful for large codebases where you want
opt-in checking for critical types only.

```go
// Only checked in explicit mode when marked
//exhaustruct:enforce
type Config struct {
    Host string
    Port int
}

// Not checked in explicit mode (unless matched by pattern)
type Options struct {
    Timeout int
}
```

## Comment Directives

Comment directives provide fine-grained control over checking behavior.
They can be placed on the line above or on the same line as the target.

### On Type Definitions

Available directives: `enforce`, `ignore`, `optional`

```go
// All literals of this type will be checked (useful in explicit mode)
//exhaustruct:enforce
type Config struct {
    Host string
    Port int
}

// All literals of this type will be skipped
//exhaustruct:ignore
type InternalState struct {
    cache map[string]any
}

// All fields of this type are optional
//exhaustruct:optional
type Options struct {
    Timeout  int
    MaxConns int
}
```

### On Struct Literals

Available directives: `enforce`, `ignore`

```go
func example() {
    //exhaustruct:ignore — skip this specific literal
    _ = Config{}

    _ = Config{} //exhaustruct:ignore — inline form also works

    //exhaustruct:enforce — check even if type is normally ignored
    _ = InternalState{}
}
```

### On Fields

Available directives: `optional`, `enforce`

```go
type Server struct {
    Host string
    Port int

    //exhaustruct:optional — this field is not required
    Timeout int

    MaxConns int //exhaustruct:optional — inline form also works

    //exhaustruct:enforce — required even if type is marked optional
    Logger Logger
}
```

### Directive Priority

When multiple directives or patterns apply, priority is (highest first):

1. Literal `//exhaustruct:ignore`
2. Literal `//exhaustruct:enforce`
3. Type-level ignore (directive or `-ignore-rx` pattern)
4. Type-level enforce (directive or `-enforce-rx` pattern)
5. Mode default (implicit=check, explicit=skip)

## Configuration

### Type Selection Flags

| Flag | Description |
|------|-------------|
| `-explicit` | Enable explicit mode (opt-in checking) |
| `-enforce-rx` | Regex pattern for types/fields to check (repeatable) |
| `-ignore-rx` | Regex pattern for types to skip (repeatable) |
| `-optional-rx` | Regex pattern for types/fields to mark optional (repeatable) |

### Empty Literal Allowances

| Flag | Description |
|------|-------------|
| `-allow-empty` | Allow all empty struct literals globally |
| `-allow-empty-rx` | Regex pattern for types allowed to be empty (repeatable) |
| `-allow-empty-returns` | Allow empty literals in return statements |
| `-allow-empty-declarations` | Allow empty literals in `var` and `:=` declarations |

### Output Flags

| Flag | Description |
|------|-------------|
| `-report-full-type-path` | Show full package path in errors (e.g., `net/http.Cookie`) |
| `-debug-cache-metrics` | Print cache statistics to stderr |

### Pattern Format

All regex patterns (`-*-rx` flags) match against **full paths**.

For types:
```
package/path.TypeName
```

For fields:
```
package/path.TypeName#FieldName
```

For anonymous structs, use `<anonymous>` as the type name:
```
package/path.<anonymous>
```

Examples:
- `net/http\.Request` — matches type `http.Request`
- `.*\.Config` — matches any type named `Config`
- `.*\.Server#Timeout` — matches field `Timeout` in any `Server` type
- `github\.com/user/repo/pkg\..*` — matches all types in a package
- `.*\.<anonymous>` — matches all anonymous structs
- `mypackage\.<anonymous>#Field` — matches field `Field` in anonymous structs in `mypackage`

## Special Behaviors

### Error Returns

Empty struct literals are automatically allowed in return statements when
accompanied by a non-nil error value:

```go
func LoadConfig() (Config, error) {
    if err := validate(); err != nil {
        return Config{}, err // OK: error return
    }
    return Config{Host: "localhost", Port: 8080}, nil
}
```

### Unexported Fields

Fields that are unexported and belong to external packages are never required,
as they cannot be initialized from outside the package:

```go
import "external/pkg"

func example() {
    // If pkg.Server has unexported fields, they are not required
    _ = pkg.Server{Host: "localhost"} // OK
}
```

### Derived Types and Aliases

Type aliases and derived types inherit **field-level** directives from their
underlying struct, but **type-level** directives are not inherited:

```go
//exhaustruct:enforce
type Config struct {
    Host string
    //exhaustruct:optional
    Timeout int
}

type MyConfig = Config      // alias: inherits optional Timeout, but NOT enforce
type ExtConfig Config       // derived: inherits optional Timeout, but NOT enforce

func example() {
    // Config is enforced (has type-level directive)
    _ = Config{} // ERROR in explicit mode

    // MyConfig inherits field optionality but not type enforcement
    _ = MyConfig{Host: "localhost"} // OK: Timeout is optional (inherited)
    _ = MyConfig{}                  // OK in explicit mode: type not enforced
}
```

To enforce checking on derived or aliased types, add directives on their
definitions:

```go
//exhaustruct:enforce
type StrictConfig = Config  // now enforced independently

//exhaustruct:enforce
type StrictExtConfig Config // now enforced independently
```

Field-level directives (`//exhaustruct:optional`, `//exhaustruct:enforce` on fields)
apply to the struct's field definitions and are shared by all types using that
underlying struct. Type-level directives control whether literals of that
specific type are checked and must be specified separately for each type.

## Migration from v4

### New Features in v5

- **Explicit mode** (`-explicit`): Opt-in checking instead of check-all
- **Optional patterns** (`-optional-rx`): Mark all fields of matching types
  as optional
- **Field patterns**: `-enforce-rx` and `-optional-rx` can now match individual
  fields using `Type#Field` syntax
- **Type-level directives**: `//exhaustruct:enforce`, `//exhaustruct:ignore`,
  and `//exhaustruct:optional` can be placed on type definitions
- **Field-level enforce**: `//exhaustruct:enforce` on fields forces them to be
  required even when the type is optional

### Flag Renames

| v4 | v5 |
|----|-----|
| `-include-rx` / `-i` | `-enforce-rx` |
| `-exclude-rx` / `-e` | `-ignore-rx` |

### Struct Tags Deprecated

Struct tags like `exhaustruct:"optional"` are no longer supported. Use comment
directives instead:

```go
// v4 (deprecated)
type Server struct {
    Host    string
    Timeout int `exhaustruct:"optional"`
}

// v5
type Server struct {
    Host    string
    //exhaustruct:optional
    Timeout int
}
```

Run with `-fix` to automatically migrate struct tags to comment directives:

```shell
exhaustruct -fix ./...
```
