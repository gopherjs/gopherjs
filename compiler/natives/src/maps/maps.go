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
	// TL:DR; A simple Go copy is faster when less than 7 pairs, however
	// since many maps may be small we switch betwen the two algorithms.
	if len(m) < 7 {
		mcopy := make(M, len(m))
		for k, v := range m {
			mcopy[k] = v
		}
		return mcopy
	}

	mptr := &M{}
	cloned := js.Global.Get("Map").New(js.InternalObject(m))
	js.InternalObject(mptr).Call(`$set`, cloned)
	return *mptr
}
