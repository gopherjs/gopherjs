//go:build js

package subtle

import "github.com/gopherjs/gopherjs/js"

const wordSize = 4 // bytes for a Uint32Array

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

	// The original uses unsafe and uintptr for specific architecture
	// to pack registers full instead of doing one byte at a time.
	// We can't do the unsafe conversions from []byte to []uintptr
	// but we can convert a Uint8Array into a Uint32Array,
	// so we'll simply do it four bytes at a time plus any remainder.
	// The following is similar to xorBytes from xor_generic.go

	dst = dst[:n]
	x = x[:n]
	y = y[:n]
	if wordCount := n / wordSize; wordCount > 0 &&
		aligned(dst) && aligned(x) && aligned(y) {
		dstWords := words(dst)
		xWords := words(x)
		yWords := words(y)
		for i := range dstWords {
			dstWords[i] = xWords[i] ^ yWords[i]
		}
		done := n &^ int(wordSize-1)
		dst = dst[done:]
		x = x[done:]
		y = y[done:]
	}
	for i := range dst {
		dst[i] = x[i] ^ y[i]
	}
	return n
}

// aligned determines whether the slice is word-aligned since
// Uint32Array's require the offset to be multiples of 4.
func aligned(b []byte) bool {
	slice := js.InternalObject(b)
	offset := slice.Get(`$offset`).Int()
	return offset%wordSize == 0
}

// words returns a []uint pointing at the same data as b,
// with any trailing partial word removed.
// The given b must have a word aligned offset.
func words(b []byte) []uint {
	slice := js.InternalObject(b)
	offset := slice.Get(`$offset`).Int()
	length := slice.Get(`$length`).Int()
	byteBuffer := slice.Get(`$array`).Get(`buffer`)
	wordBuffer := js.Global.Get(`Uint32Array`).New(byteBuffer, offset, length/wordSize)
	return wordBuffer.Interface().([]uint)
}

//gopherjs:purge
const supportsUnaligned = false

//gopherjs:purge
func xorBytes(dstb, xb, yb *byte, n int)

// TODO(grantnelson-wf): Check if this should be removed or not with generics.
//
//gopherjs:purge
func xorLoop[T byte | uintptr](dst, x, y []T) {}
