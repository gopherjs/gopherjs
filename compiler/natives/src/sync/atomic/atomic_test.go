//go:build js
// +build js

package atomic_test

import (
	"runtime"
	"strings"
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

func TestHammer32(t *testing.T) { // TODO: REMOVE
	const p = 4
	n := 100000
	if testing.Short() {
		n = 1000
	}
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(p))

	for name, testf := range hammer32 {
		c := make(chan int)
		var val uint32
		for i := 0; i < p; i++ {
			go func() {
				defer func() {
					if err := recover(); err != nil {
						t.Error(err)
					}
					c <- 1
				}()
				testf(&val, n)
			}()
		}
		for i := 0; i < p; i++ {
			<-c
		}
		if !strings.HasPrefix(name, "Swap") && val != uint32(n)*p {
			t.Fatalf("%s: val=%d want %d", name, val, n*p)
		}
	}
}

func TestHammer64(t *testing.T) { // TODO: REMOVE
	if test64err != nil {
		t.Skipf("Skipping 64-bit tests: %v", test64err)
	}
	const p = 4
	n := 100000
	if testing.Short() {
		n = 1000
	}
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(p))

	for name, testf := range hammer64 {
		c := make(chan int)
		var val uint64
		for i := 0; i < p; i++ {
			go func() {
				defer func() {
					if err := recover(); err != nil {
						t.Error(err)
					}
					c <- 1
				}()
				testf(&val, n)
			}()
		}
		for i := 0; i < p; i++ {
			<-c
		}
		if !strings.HasPrefix(name, "Swap") && val != uint64(n)*p {
			t.Fatalf("%s: val=%d want %d", name, val, n*p)
		}
	}
}
