//go:build js

package unsafeheader_test

import "testing"

func TestTypeMatchesReflectType(t *testing.T) {
	t.Skip("GopherJS uses different slice and string implementation than internal/unsafeheader.")
}

//gopherjs:purge
func testHeaderMatchesReflect()

//gopherjs:purge
func typeCompatible()

func TestWriteThroughHeader(t *testing.T) {
	t.Skip("GopherJS uses different slice and string implementation than internal/unsafeheader.")
}
