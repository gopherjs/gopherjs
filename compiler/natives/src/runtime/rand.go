//go:build js

package runtime

import "github.com/gopherjs/gopherjs/js"

func rand32() uint32 {
	return uint32(rand32f())
}

//go:linkname rand
func rand() uint64 {
	return js.MakeUint64(rand32f(), rand32f())
}

//go:linkname randn
func randn(n uint32) uint32 {
	return legacy_fastrand() % n
}

// This returns a JS float filled with a randomized uint32 value.
//
//gopherjs:new
func rand32f() float64 {
	math := js.Global.Get(`Math`)
	return math.Call("floor", math.Call("random").Float()*(1<<32-1)).Float()
}

func legacy_fastrand() uint32 {
	return rand32()
}

func legacy_fastrandn(n uint32) uint32 {
	return randn(n)
}

func legacy_fastrand64() uint64 {
	return rand()
}

func fastrandu() uint {
	return uint(legacy_fastrand())
}
