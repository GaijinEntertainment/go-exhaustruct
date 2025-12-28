# Test Data Structure

This directory contains test fixtures for the exhaustruct analyzer, organized as a standalone Go module.

## Philosophy

### Standalone Module

The testdata directory is a separate Go module (`module testdata`) which provides:

- **IDE Support**: Files are navigable and type-checkable in IDEs without running tests
- **Import Clarity**: Cross-package tests use real import paths (`testdata/external`)
- **Isolation**: Test types don't pollute the main module namespace

### Package Organization

Tests are organized by **behavior category**, not by implementation detail:

```
testdata/
├── external/                  # Common types for cross-package testing
├── types/                     # Core type behavior tests
│   ├── basic/                 # Basic struct literal tests
│   ├── derived/               # Derived types (type T Base)
│   ├── aliases/               # Type aliases (type T = Base)
│   ├── embedded/              # Embedded struct fields
│   ├── generics/              # Generic struct types
│   ├── collections/           # Structs in slices and maps
│   ├── anonymous/             # Anonymous struct types
│   ├── directives/            # Comment directives (ignore/enforce)
│   └── filtering/             # Include/exclude pattern tests
└── config/                    # Configuration option tests
    ├── excluded/              # ExcludeRx pattern behavior
    ├── report_full_path/      # ReportFullTypePath=true
    └── allow_empty/           # Empty struct allowance tests
        ├── global/            # AllowEmpty=true
        ├── returns/           # AllowEmptyReturns=true
        ├── declarations/      # AllowEmptyDeclarations=true
        ├── patterns/          # AllowEmptyRx patterns
        └── error_returns/     # Error return behavior
```

Each subpackage is **self-contained**:
- Defines its own local types
- Imports from `external/` for cross-package scenarios
- Has dedicated test configuration in `analyzer_test.go`

### Local vs External Types

Testing struct field behavior requires two contexts:

1. **Local types**: Defined in the same package as the literal
   - Unexported fields ARE accessible
   - All fields should be checked

2. **External types**: Imported from another package
   - Unexported fields are NOT accessible
   - Only exported fields should be checked

The `external/` package provides common types with unexported fields specifically for testing this distinction.

### Issue-Specific Tests

Bug reproductions and fixes are documented in dedicated files:

```
types/derived/
├── derived.go      # Core derived type behavior
└── issue_xxx.go    # Example: bug-specific test (e.g., external derived types + unexported fields)
```

This approach:
- **Documents bugs** with context and links
- **Prevents regressions** when bugs are fixed
- **Separates concerns** between expected behavior and known issues

### Test Configuration

Each subpackage may need different analyzer configuration. Tests use table-driven approach:

```go
func TestAnalyzerTypes(t *testing.T) {
    tests := []struct {
        name        string
        config      analyzer.Config
        testPackage string
    }{
        {
            name: "derived types",
            config: analyzer.Config{
                IncludeRx: []string{`.*\.(Base|Derived).*`},
                ExcludeRx: []string{`.*Excluded.*`},
            },
            testPackage: "testdata/types/derived",
        },
    }
    // ...
}
```

This allows running individual test suites: `go test -run TestAnalyzerTypes/derived_types`

## Naming Conventions

### Types

- `Base`, `Simple` - Base types for deriving/aliasing
- `*Derived` - Types created with `type T Base`
- `*Alias` - Types created with `type T = Base`
- `*Excluded` - Types matching exclusion patterns
- `External*` - Types derived from external package

### Functions

- `shouldPass*` - No `// want` comment, expects no diagnostic
- `shouldFail*` - Has `// want` comment, expects diagnostic

### Files

- `<category>.go` - Core behavior tests
- `issue_<number>.go` - Bug reproduction/regression tests
