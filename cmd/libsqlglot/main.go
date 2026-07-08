// Command libsqlglot is the C-shared entry point for the JVM binding (jvm/): it exports the probe
// as a C ABI function so a JVM process can call it in-process via the Foreign Function & Memory API
// (java.lang.foreign). It is a separate `main` package, so cgo is confined here — pure-Go consumers
// of the library never pull in cgo.
//
// Build:  go build -buildmode=c-shared -o libsqlglot.<dylib|so> ./cmd/libsqlglot
package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"

	"github.com/sjincho/sqlglot-go/probe"
)

// ProbeJSON analyzes one SQL statement and returns the ProbeResult contract as a JSON string.
// It mirrors probe.py's probe_json(sql, dialect, schema_json): dialect is "mysql"|"postgres",
// schemaJSON is {"table":{"column":"TYPE"}}. The returned char* is malloc'd by Go and MUST be
// released by the caller via FreeCString. probe.ProbeJSONSafe is total (never panics, always valid
// JSON), so this boundary can never crash the host JVM.
//
//export ProbeJSON
func ProbeJSON(sql, dialect, schemaJSON *C.char) *C.char {
	out := probe.ProbeJSONSafe(C.GoString(sql), C.GoString(dialect), C.GoString(schemaJSON))
	return C.CString(out) // C.CString mallocs; caller frees via FreeCString.
}

// FreeCString releases a char* previously returned by ProbeJSON.
//
//export FreeCString
func FreeCString(p *C.char) {
	C.free(unsafe.Pointer(p))
}

func main() {}
