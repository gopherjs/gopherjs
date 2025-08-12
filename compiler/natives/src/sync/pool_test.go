//go:build js

package sync_test

import (
	. "sync"
	"testing"
)

//gopherjs:replace
func TestPool(t *testing.T) {
	var p Pool
	if p.Get() != nil {
		t.Fatal("expected empty")
	}

	p.Put("a")
	p.Put("b")

	want := []interface{}{"b", "a", nil}
	for i := range want {
		got := p.Get()
		if got != want[i] {
			t.Fatalf("Got: p.Get() returned: %s. Want: %s.", got, want)
		}
	}
}

//gopherjs:replace
func TestPoolGC(t *testing.T) {
	t.Skip("This test uses runtime.GC(), which GopherJS doesn't support.")
}

//gopherjs:replace
func TestPoolRelease(t *testing.T) {
	t.Skip("This test uses runtime.GC(), which GopherJS doesn't support.")
}

//gopherjs:replace
func TestPoolDequeue(t *testing.T) {
	t.Skip("This test targets upstream pool implementation, which is not used by GopherJS.")
}

//gopherjs:replace
func TestPoolChain(t *testing.T) {
	t.Skip("This test targets upstream pool implementation, which is not used by GopherJS.")
}
