//go:build js
// +build js

package rand

//go:linkname fastrand64 runtime.fastrand64
func fastrand64() uint64
