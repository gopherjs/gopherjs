//go:build js

package reflect

import (
	"unsafe"

	"internal/abi"

	"github.com/gopherjs/gopherjs/compiler/natives/src/internal/unsafeheader"
)

//gopherjs:purge This is the header for an any interface and invalid for GopherJS.
type emptyInterface struct{}

//gopherjs:purge This is the header for an interface value with methods and invalid for GopherJS.
type nonEmptyInterface struct{}

//gopherjs:purge
func packEface(v Value) any

//gopherjs:purge
func unpackEface(i any) Value

//gopherjs:purge
func storeRcvr(v Value, p unsafe.Pointer)

//gopherjs:purge
func callMethod(ctxt *methodValue, frame unsafe.Pointer, retValid *bool, regs *abi.RegArgs)

// typedslicecopy is implemented in prelude.js as $copySlice
//
//gopherjs:purge
func typedslicecopy(t *abi.Type, dst, src unsafeheader.Slice) int

// growslice is implemented in prelude.js as $growSlice.
//
//gopherjs:purge
func growslice(t *abi.Type, old unsafeheader.Slice, num int) unsafeheader.Slice

// gopherjs:replace
func noescape(p unsafe.Pointer) unsafe.Pointer {
	return p
}

//gopherjs:purge
func makeFuncStub()
