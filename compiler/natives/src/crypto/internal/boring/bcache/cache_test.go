//go:build js

package bcache

import "testing"

var registeredCache Cache

func TestCache(t *testing.T) {
	t.Skip(`This test uses runtime.GC(), which GopherJS doesn't support`)
}
