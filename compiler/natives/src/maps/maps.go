//go:build js

package maps

import "github.com/gopherjs/gopherjs/js"

// Clone returns a copy of m.  This is a shallow clone:
// the new keys and values are set using ordinary assignment.
//
//gopherjs:replace
func Clone[M ~map[K]V, K comparable, V any](m M) M {
	if m == nil {
		return nil
	}

	// See benchmark in ./tests/map_js_test.go
	mptr := &M{}
	cloned := js.Global.Get("Map").New(js.InternalObject(m))
	js.InternalObject(mptr).Call(`$set`, cloned)
	return *mptr
}
