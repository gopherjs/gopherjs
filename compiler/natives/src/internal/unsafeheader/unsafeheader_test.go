//go:build js
// +build js

package unsafeheader_test

import "testing"

//gopherjs:replace
func TestWriteThroughHeader(t *testing.T) {
	t.Skip("GopherJS uses different slice and string implementation than internal/unsafeheader.")
}
