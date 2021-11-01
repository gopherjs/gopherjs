//go:build js
// +build js

package big

import "testing"

func TestBytes(t *testing.T) {
	t.Skip("broken")
}

func TestModSqrt(t *testing.T) {
	t.Skip("slow")
}

func TestLinkerGC(t *testing.T) {
	t.Skip("The test is specific to GC's linker.")
}
