//go:build js && wasm
// +build js,wasm

package http_test

import (
	"testing"
)

func testTransportGCRequest(t *testing.T, mode testMode, body bool) {
	t.Skip("The test relies on runtime.SetFinalizer(), which is not supported by GopherJS.")
}

func testWriteHeaderAfterWrite(t *testing.T, mode testMode, hijack bool) {
	t.Skip("GopherJS source maps don't preserve original function names in stack traces, which this test relied on.")
}
