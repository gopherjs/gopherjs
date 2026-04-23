//go:build js && wasm

package bytes_test

import (
	"testing"
)

//gopherjs:replace
func TestIssue65571(t *testing.T) {
	t.Skip("TestIssue65571 expects `int` to be greater than 32 bits")
	// This fails with `1 << 31 + 1 (untyped int constant 2147483649) overflows int`
	// since int is set to 32 bits when performing the type check even though
	// JS can handle larger values during runtime.
}
