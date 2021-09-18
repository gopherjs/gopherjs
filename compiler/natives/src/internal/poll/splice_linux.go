//go:build js
// +build js

package poll

import "unsafe"

// Workaround for https://github.com/gopherjs/gopherjs/issues/1060.
var disableSplice unsafe.Pointer = unsafe.Pointer((*bool)(nil))
