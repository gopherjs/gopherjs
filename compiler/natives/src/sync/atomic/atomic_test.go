//go:build js

package atomic_test

import "testing"

//gopherjs:replace
func TestHammerStoreLoad(t *testing.T) {
	t.Skip("use of unsafe")
}

//gopherjs:replace
func TestUnaligned64(t *testing.T) {
	t.Skip("GopherJS emulates atomics, which makes alignment irrelevant.")
}

//gopherjs:replace
func TestAutoAligned64(t *testing.T) {
	t.Skip("GopherJS emulates atomics, which makes alignment irrelevant.")
}

//gopherjs:replace
func TestHammer32(t *testing.T) {
	t.Skip("use of unsafe")
}

//gopherjs:replace
func TestHammer64(t *testing.T) {
	t.Skip("use of unsafe")
}
