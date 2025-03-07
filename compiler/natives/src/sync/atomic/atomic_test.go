//go:build js
// +build js

package atomic_test

import (
	"sync/atomic"
	"testing"
)

func TestHammerStoreLoad(t *testing.T) {
	t.Skip("use of unsafe")
}

func TestUnaligned64(t *testing.T) {
	t.Skip("GopherJS emulates atomics, which makes alignment irrelevant.")
}

func TestAutoAligned64(t *testing.T) {
	t.Skip("GopherJS emulates atomics, which makes alignment irrelevant.")
}

func TestHammer32(t *testing.T) {
	t.Skip("use of unsafe")
}

func TestHammer64(t *testing.T) {
	t.Skip("use of unsafe")
}

func TestSwapPointerMethod(t *testing.T) {
	var x struct {
		before uintptr
		i      atomic.Pointer[byte]
		after  uintptr
	}
	var m uint64 = magic64
	magicptr := uintptr(m)
	x.before = magicptr
	x.after = magicptr

	// TODO(grantnelson-wf): We need to fix the initialization of unsafe.Pointer
	// or make an uninitialized equal to nil(nil).
	//
	// This tests fails when `k != j` because `k` is set to the x.i.v which is not
	// initialized and set to undefined whilst `j` is set to nil, so they are not
	// equal when they should be. Then the undefined `k` causes a `panic` when
	// trying to be printed via t.Fatalf because it can't get "length" from undefined.
	//
	// To fix this test, force initialize x.i.v to nil, to prevent it from being undefined.
	x.i.Store(nil)

	var j *byte
	for _, p := range testPointers() {
		p := (*byte)(p)
		k := x.i.Swap(p)
		if x.i.Load() != p || k != j {
			t.Fatalf("p=%p i=%p j=%p k=%p", p, x.i.Load(), j, k)
		}
		j = p
	}
	if x.before != magicptr || x.after != magicptr {
		t.Fatalf("wrong magic: %#x _ %#x != %#x _ %#x", x.before, x.after, magicptr, magicptr)
	}
}
