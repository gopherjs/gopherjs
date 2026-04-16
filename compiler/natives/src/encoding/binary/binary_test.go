//go:build js

package binary

import (
	"testing"
)

//gopherjs:replace The original cast the uint32 into a *byte which doesn't work in JS
func TestNativeEndian(t *testing.T) {
	const val = 0x12345678
	s := []byte{0x78, 0x56, 0x34, 0x12} // LittleEndian
	if v := NativeEndian.Uint32(s); v != val {
		t.Errorf("NativeEndian.Uint32 returned %#x, expected %#x", v, val)
	}
}
