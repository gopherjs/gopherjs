//go:build js

package slices

import "github.com/gopherjs/gopherjs/js"

//gopherjs:replace
func overlaps[E any](a, b []E) bool {
	// GopherJS: We can't rely on pointer arithmetic, so use GopherJS slice internals.
	return len(a) > 0 && len(b) > 0 &&
		js.InternalObject(a).Get("$array") == js.InternalObject(b).Get("$array") &&
		js.InternalObject(a).Get("$offset").Int() <= js.InternalObject(b).Get("$offset").Int()+len(b)-1 &&
		js.InternalObject(b).Get("$offset").Int() <= js.InternalObject(a).Get("$offset").Int()+len(a)-1
}
