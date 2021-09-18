//go:build js
// +build js

package js

// Defined to avoid a compile error in the original TestGarbageCollection()
// body. Can't use gopherjs:prune-original on it, since it causes an unused
// import error.
var JSGo Value
