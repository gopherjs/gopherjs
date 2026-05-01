//go:build js

package maps

// Clone returns a copy of m.  This is a shallow clone:
// the new keys and values are set using ordinary assignment.
//
//gopherjs:replace
func Clone[M ~map[K]V, K comparable, V any](m M) M {
	if m == nil {
		return nil
	}

	// A simple Go copy version of clone may be slower for large maps
	// and faster for small maps than using the JS Map constructor to create
	// a copy. However, the Go copy ensures that we don't run into a
	// potentical issue with JS objects not being properly copied during clone.
	mcopy := make(M, len(m))
	for k, v := range m {
		mcopy[k] = v
	}
	return mcopy
}
