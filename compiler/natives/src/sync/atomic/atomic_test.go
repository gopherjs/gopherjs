//go:build js
// +build js

package atomic_test

import "testing"

func TestHammerStoreLoad(t *testing.T) {
	t.Skip("use of unsafe")
}

func TestUnaligned64(t *testing.T) {
	t.Skip("GopherJS emulates atomics, which makes alignment irrelevant.")
}
