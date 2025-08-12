package debug_test

import "testing"

//gopherjs:replace
func TestReadGCStats(t *testing.T) {
	t.Skip(`This test uses runtime.GC(), which GopherJS doesn't support`)
}

//gopherjs:replace
func TestFreeOSMemory(t *testing.T) {
	t.Skip(`This test uses runtime.GC(), which GopherJS doesn't support`)
}
