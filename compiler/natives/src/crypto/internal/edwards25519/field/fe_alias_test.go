//go:build js

package field

import (
	"testing"
	"testing/quick"
)

//gopherjs:keep-original
func TestAliasing(t *testing.T) {
	// The test heavily uses 64-bit math, which is slow under GopherJS. Reducing
	// the number of iterations makes run time more manageable.
	t.Cleanup(quick.GopherJSInternalMaxCountCap(100))
	_gopherjs_original_TestAliasing(t)
}
