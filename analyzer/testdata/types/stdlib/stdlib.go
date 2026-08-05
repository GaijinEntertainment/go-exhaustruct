// Package stdlib exercises literals of standard library types.
//
// Standard library sources carry no exhaustruct directives, so their
// definitions are resolved as directive-free. That applies to directives only:
// the structs themselves are checked like any other.
//
// Unlike the other fixtures, this one is not driven by analysistest, which
// type-checks dependencies from source. It needs them loaded from export data,
// where standard library positions carry the compiler's literal "$GOROOT"
// placeholder. See TestAnalyzer_StdlibExportData.
package stdlib

import (
	"net"
	"strings"
	"sync"
)

func shouldFailEmpty() {
	_ = net.TCPAddr{}
}

func shouldFailPartial() {
	_ = net.TCPAddr{IP: nil}
}

func shouldPassFullyDefined() {
	_ = net.TCPAddr{IP: nil, Port: 0, Zone: ""}
}

// Unexported fields of another package are not required, so these literals are
// reported by neither this analyzer nor the compiler.
func shouldPassUnexportedOnly() {
	_ = strings.Builder{}
	_ = sync.Mutex{}
}
