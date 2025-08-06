package maphash

import "testing"

//gopherjs:keep-original
func TestSmhasherSmallKeys(t *testing.T) {
	if !testing.Short() {
		t.Skip("Causes a heap overflow in GopherJS when not --short")
		// This test adds a lot of uint64 hashes into a map,
		// (16,843,008 for long tests, 65,792 for short tests)
		// inside `(s *hashSet) add(h uint64)` with `s.m[h] = struct{}{}`.
		// This is to check the number of collisions in the hash function.
	}
	_gopherjs_original_TestSmhasherSmallKeys(t)
}

//gopherjs:keep-original
func TestSmhasherZeros(t *testing.T) {
	if !testing.Short() {
		t.Skip("Too slow when not --short")
		// This test creates a byte slice with 262,144 bytes for long tests
		// and 1,024 for short tests filled by defualt with zeroes.
		// Then it adds [:1], [:2], and so on upto the full slice.
	}
	_gopherjs_original_TestSmhasherZeros(t)
}

//gopherjs:replace
func TestSmhasherTwoNonzero(t *testing.T) {
	// The original skips if `runtime.GOARCH == "wasm"` which means we should skip too.
	t.Skip("Too slow on wasm and JS")
}

//gopherjs:replace
func TestSmhasherSparse(t *testing.T) {
	// The original skips if `runtime.GOARCH == "wasm"` which means we should skip too.
	t.Skip("Too slow on wasm and JS")
}

//gopherjs:replace
func TestSmhasherPermutation(t *testing.T) {
	// The original skips if `runtime.GOARCH == "wasm"` which means we should skip too.
	t.Skip("Too slow on wasm and JS")
}

//gopherjs:replace
func TestSmhasherAvalanche(t *testing.T) {
	// The original skips if `runtime.GOARCH == "wasm"` which means we should skip too.
	t.Skip("Too slow on wasm and JS")
}

//gopherjs:replace
func TestSmhasherWindowed(t *testing.T) {
	// The original skips if `runtime.GOARCH == "wasm"` which means we should skip too.
	t.Skip("Too slow on wasm and JS")
}
