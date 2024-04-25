//go:build js
// +build js

package bcache

import "testing"

//gopherjs:purge
var registeredCache Cache

func TestCache(t *testing.T) {
	t.Skip(`This test uses runtime.GC(), which GopherJS doesn't support`)
}
