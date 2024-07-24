//go:build js

package gif

import "testing"

//gopherjs:keep-original
func FuzzDecode(t *testing.F) {
	if testing.Short() {
		t.Skip("FuzzDecode is slow, skipping in the short mode.")
	}

	_gopherjs_original_FuzzDecode(t)
}
