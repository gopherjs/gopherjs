//go:build js
// +build js

package js_test

import "testing"

//gopherjs:prune-original
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
