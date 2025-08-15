//go:build js

package atomic_test

import (
	"testing"
	"unsafe"
)

//gopherjs:purge for go1.19 without generics
func testPointers() []unsafe.Pointer {}

func TestSwapPointer(t *testing.T) {
	t.Skip("GopherJS does not support generics yet.")
}

func TestSwapPointerMethod(t *testing.T) {
	t.Skip("GopherJS does not support generics yet.")
}

func TestCompareAndSwapPointer(t *testing.T) {
	t.Skip("GopherJS does not support generics yet.")
}

func TestCompareAndSwapPointerMethod(t *testing.T) {
	t.Skip("GopherJS does not support generics yet.")
}

func TestLoadPointer(t *testing.T) {
	t.Skip("GopherJS does not support generics yet.")
}

func TestLoadPointerMethod(t *testing.T) {
	t.Skip("GopherJS does not support generics yet.")
}

func TestStorePointer(t *testing.T) {
	t.Skip("GopherJS does not support generics yet.")
}

func TestStorePointerMethod(t *testing.T) {
	t.Skip("GopherJS does not support generics yet.")
}

//gopherjs:purge for go1.19 without generics
func hammerStoreLoadPointer(t *testing.T, paddr unsafe.Pointer) {}

//gopherjs:purge for go1.19 without generics
func hammerStoreLoadPointerMethod(t *testing.T, paddr unsafe.Pointer) {}

func TestHammerStoreLoad(t *testing.T) {
	t.Skip("use of unsafe")
}

func TestUnaligned64(t *testing.T) {
	t.Skip("GopherJS emulates atomics, which makes alignment irrelevant.")
}

func TestAutoAligned64(t *testing.T) {
	t.Skip("GopherJS emulates atomics, which makes alignment irrelevant.")
}

func TestNilDeref(t *testing.T) {
	t.Skip("GopherJS does not support generics yet.")
}

//gopherjs:purge for go1.19 without generics
type List struct{}

func TestHammer32(t *testing.T) {
	t.Skip("use of unsafe")
}

func TestHammer64(t *testing.T) {
	t.Skip("use of unsafe")
}
