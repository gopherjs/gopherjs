//go:build js

package cpu

//gopherjs:replace
const (
	CacheLineSize    = 0
	CacheLinePadSize = 0
)

//gopherjs:replace
func doinit() {}
