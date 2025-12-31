// Package fileutil provides file reading utilities for the analyzer.
package fileutil

import (
	"os"

	"dev.gaijin.team/go/golib/e"
	"dev.gaijin.team/go/golib/fields"
)

// ReadFileFunc is a function that reads file content by filename.
type ReadFileFunc func(filename string) ([]byte, error)

// Reader provides file reading with optional primary function and OS fallback.
type Reader struct {
	primary ReadFileFunc
}

// NewReader creates a Reader with optional primary read function.
// If primary is nil, only os.ReadFile will be used.
// If primary is provided but fails, falls back to os.ReadFile.
func NewReader(primary ReadFileFunc) *Reader {
	return &Reader{primary: primary}
}

// ReadFile reads file content by filename.
// Tries primary function first (if set), falls back to os.ReadFile on nil or error.
func (r *Reader) ReadFile(filename string) ([]byte, error) {
	if r.primary != nil {
		content, err := r.primary(filename)
		if err == nil {
			return content, nil
		}
		// primary failed, fall back to OS
	}

	content, err := os.ReadFile(filename) //nolint:gosec // filename comes from token.FileSet, not user input
	if err != nil {
		return nil, e.NewFrom("read file", err, fields.F("filename", filename))
	}

	return content, nil
}
