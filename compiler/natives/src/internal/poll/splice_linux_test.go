//go:build js
// +build js

package poll_test

import "testing"

func TestSplicePipePool(t *testing.T) {
	t.Skip("Test relies upon runtime.SetFinalizer and runtime.GC(), which are not supported by GopherJS.")
}
