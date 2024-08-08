//go:build js
// +build js

package atomic_test

import (
	"testing"
)

func TestSwapPointerMethod(t *testing.T) {
	t.Skip("GopherJS does not fully support generics for go1.20 yet.")
}

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
