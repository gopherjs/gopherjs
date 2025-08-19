//go:build js

package debug_test

import "testing"

func TestReadGCStats(t *testing.T) {
	t.Skip(`This test uses runtime.GC(), which GopherJS doesn't support`)
}

func TestFreeOSMemory(t *testing.T) {
	t.Skip(`This test uses runtime.GC(), which GopherJS doesn't support`)
}
