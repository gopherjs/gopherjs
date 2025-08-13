//go:build js && wasm

package http_test

import (
	"testing"
)

func testTransportGCRequest(t *testing.T, h2, body bool) {
	t.Skip("The test relies on runtime.SetFinalizer(), which is not supported by GopherJS.")
}

func testWriteHeaderAfterWrite(t *testing.T, h2, hijack bool) {
	t.Skip("GopherJS source maps don't preserve original function names in stack traces, which this test relied on.")
}
