//go:build js

package js_test

import "testing"

func TestIntConversion(t *testing.T) {
	// Same as upstream, but only test cases appropriate for a 32-bit environment.
	testIntConversion(t, 0)
	testIntConversion(t, 1)
	testIntConversion(t, -1)
	testIntConversion(t, 1<<20)
	testIntConversion(t, -1<<20)
}

func TestGarbageCollection(t *testing.T) {
	t.Skip("GC is not supported by GopherJS")
}

// GopherJS does not currently support the //go:wasmimport directive
// (see https://go.dev/blog/wasi and https://github.com/golang/go/issues/59149).
// Functions declared with //go:wasmimport compile to "native function not
// implemented" runtime stubs, which panic and crash the Node process before
// the testing framework can recover. Skip the affected tests until proper
// support is added; they cannot be overridden without defeating the test.
//
//gopherjs:replace
func TestWasmImport(t *testing.T) {
	t.Skip("//go:wasmimport is not supported by GopherJS")
}
