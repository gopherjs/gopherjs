//go:build js

package abi

import (
	"unsafe"

	"github.com/gopherjs/gopherjs/js"
)

// GOPHERJS: These utils are being added because they are common between
// reflect and reflectlite. The [Go proverb](https://go-proverbs.github.io/),
// "A little copying is better than a little dependency," isn't applicable
// when both reflect and reflectlite already depend on ABI. We can reduce
// our native overrides in both locations by putting common code here.

//gopherjs:new
func UnsafeNew(typ *Type) unsafe.Pointer {
	switch typ.Kind() {
	case Struct:
		return unsafe.Pointer(typ.JsType().Get("ptr").New().Unsafe())
	case Array:
		return unsafe.Pointer(typ.JsType().Call("zero").Unsafe())
	default:
		return unsafe.Pointer(js.Global.Call("$newDataPointer", typ.JsType().Call("zero"), typ.JsPtrTo()).Unsafe())
	}
}

//gopherjs:new
func IfaceE2I(t *Type, src any, dst unsafe.Pointer) {
	js.InternalObject(dst).Call("$set", js.InternalObject(src))
}

//gopherjs:new
func TypedMemMove(t *Type, dst, src unsafe.Pointer) {
	js.InternalObject(dst).Call("$set", js.InternalObject(src).Call("$get"))
}
