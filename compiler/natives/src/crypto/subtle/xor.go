//go:build js
// +build js

package subtle

func XORBytes(dst, x, y []byte) int {
	n := len(x)
	if len(y) < n {
		n = len(y)
	}
	if n == 0 {
		return 0
	}
	if n > len(dst) {
		panic("subtle.XORBytes: dst too short")
	}

	// xorBytes(&dst[0], &x[0], &y[0], n) // arch-specific
	// The above uses unsafe and generics for specific architecture
	// to pack registers full instead of doing one byte at a time.
	// We can't do the unsafe conversions from []byte to []uintptr
	// so we'll simply do it one byte at a time.

	x = x[:len(dst)] // remove bounds check in loop
	y = y[:len(dst)] // remove bounds check in loop
	for i := range dst {
		dst[i] = x[i] ^ y[i]
	}
	return n
}

//gopherjs:purge
const (
	wordSize          = 0
	supportsUnaligned = false
)

//gopherjs:purge
func xorBytes(dstb, xb, yb *byte, n int)

//gopherjs:purge
func aligned(dst, x, y *byte) bool

//gopherjs:purge
func words(x []byte) []uintptr

//gopherjs:purge
func xorLoop[T byte | uintptr](dst, x, y []T) {}
