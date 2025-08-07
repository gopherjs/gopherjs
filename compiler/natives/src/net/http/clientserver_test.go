//go:build js && wasm

package http_test

import (
	"testing"
)

//gopherjs:replace
func testTransportGCRequest(t *testing.T, h2, body bool) {
	t.Skip("The test relies on runtime.SetFinalizer(), which is not supported by GopherJS.")
}

//gopherjs:replace
func testWriteHeaderAfterWrite(t *testing.T, h2, hijack bool) {
	// See: https://github.com/gopherjs/gopherjs/issues/1085
	t.Skip("GopherJS source maps don't preserve original function names in stack traces, which this test relied on.")
}
