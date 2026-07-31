//go:build js

package runtime

import (
	_ "unsafe"

	"github.com/gopherjs/gopherjs/js"
)

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

//go:linkname legacy_fastrand runtime.fastrand
func legacy_fastrand() uint32 {
	return rand32()
}

//go:linkname legacy_fastrandn runtime.fastrandn
func legacy_fastrandn(n uint32) uint32 {
	return randn(n)
}

//go:linkname legacy_fastrand64 runtime.fastrand64
func legacy_fastrand64() uint64 {
	return rand()
}

//go:linkname fastrandu
func fastrandu() uint {
	return uint(legacy_fastrand())
}
