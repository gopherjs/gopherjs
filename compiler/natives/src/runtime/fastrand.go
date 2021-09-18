//go:build js
// +build js

package runtime

import "github.com/gopherjs/gopherjs/js"

func fastrand() uint32 {
	// In the upstream this function is implemented with a
	// custom algorithm that uses bit manipulation, but it is likely to be slower
	// than calling Math.random().
	// TODO(nevkontakte): We should verify that it actually is faster and has a
	// similar distribution.
	return uint32(js.Global.Get("Math").Call("random").Float() * (1<<32 - 1))
}
