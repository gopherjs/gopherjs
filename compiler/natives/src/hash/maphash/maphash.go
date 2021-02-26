// +build js
// +build go1.14

package maphash

import _ "unsafe"

//go:linkname memhash_init runtime.memhash_init
func memhash_init()

func init() {
	memhash_init()
}
