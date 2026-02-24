//go:build js

package reflectlite

import (
	"unsafe"

	"internal/abi"

	"github.com/gopherjs/gopherjs/js"
)

//gopherjs:new
func (v Value) object() *js.Object {
	if v.typ.Kind() == abi.Array || v.typ.Kind() == abi.Struct {
		return js.InternalObject(v.ptr)
	}
	if v.flag&flagIndir != 0 {
		val := js.InternalObject(v.ptr).Call("$get")
		if val != js.Global.Get("$ifaceNil") && val.Get("constructor") != jsType(v.typ) {
			switch v.typ.Kind() {
			case abi.Uint64, abi.Int64:
				val = jsType(v.typ).New(val.Get("$high"), val.Get("$low"))
			case abi.Complex64, abi.Complex128:
				val = jsType(v.typ).New(val.Get("$real"), val.Get("$imag"))
			case abi.Slice:
				if val == val.Get("constructor").Get("nil") {
					val = jsType(v.typ).Get("nil")
					break
				}
				newVal := jsType(v.typ).New(val.Get("$array"))
				newVal.Set("$offset", val.Get("$offset"))
				newVal.Set("$length", val.Get("$length"))
				newVal.Set("$capacity", val.Get("$capacity"))
				val = newVal
			}
		}
		return js.InternalObject(val.Unsafe())
	}
	return js.InternalObject(v.ptr)
}

//gopherjs:replace
func (v Value) assignTo(context string, dst *abi.Type, target unsafe.Pointer) Value {
	if v.flag&flagMethod != 0 {
		v = makeMethodValue(context, v)
	}
	switch {
	case directlyAssignable(dst, v.typ):
		// Overwrite type so that they match.
		// Same memory layout, so no harm done.
		fl := v.flag&(flagAddr|flagIndir) | v.flag.ro()
		fl |= flag(dst.Kind())
		return Value{dst, v.ptr, fl}

	case implements(dst, v.typ):
		if target == nil {
			target = unsafe_New(dst)
		}
		// GopherJS: Skip the v.Kind() == Interface && v.IsNil() if statement
		//           from upstream. ifaceE2I below does not panic, and it needs
		//           to run, given its custom implementation.
		x := valueInterface(v)
		if dst.NumMethod() == 0 {
			*(*any)(target) = x
		} else {
			ifaceE2I(dst, x, target)
		}
		return Value{dst, target, flagIndir | flag(Interface)}
	}

	// Failed.
	panic(context + ": value of type " + v.typ.String() + " is not assignable to type " + dst.String())
}

//gopherjs:replace
func (v Value) IsNil() bool {
	switch k := v.kind(); k {
	case abi.Pointer, abi.Slice:
		return v.object() == jsType(v.typ).Get("nil")
	case abi.Chan:
		return v.object() == js.Global.Get("$chanNil")
	case abi.Func:
		return v.object() == js.Global.Get("$throwNilPointerError")
	case abi.Map:
		return v.object() == js.InternalObject(false)
	case abi.Interface:
		return v.object() == js.Global.Get("$ifaceNil")
	case abi.UnsafePointer:
		return v.object().Unsafe() == 0
	default:
		panic(&ValueError{"reflect.Value.IsNil", k})
	}
}

//gopherjs:replace
func (v Value) Len() int {
	switch k := v.kind(); k {
	case abi.Array, abi.String:
		return v.object().Length()
	case abi.Slice:
		return v.object().Get("$length").Int()
	case abi.Chan:
		return v.object().Get("$buffer").Get("length").Int()
	case abi.Map:
		return v.object().Get("size").Int()
	default:
		panic(&ValueError{"reflect.Value.Len", k})
	}
}

//gopherjs:replace
func (v Value) Set(x Value) {
	v.mustBeAssignable()
	x.mustBeExported()
	x = x.assignTo("reflect.Set", v.typ, nil)
	if v.flag&flagIndir != 0 {
		switch v.typ.Kind() {
		case abi.Array:
			jsType(v.typ).Call("copy", js.InternalObject(v.ptr), js.InternalObject(x.ptr))
		case abi.Interface:
			js.InternalObject(v.ptr).Call("$set", js.InternalObject(valueInterface(x)))
		case abi.Struct:
			abi.CopyStruct(js.InternalObject(v.ptr), js.InternalObject(x.ptr), v.typ)
		default:
			js.InternalObject(v.ptr).Call("$set", x.object())
		}
		return
	}
	v.ptr = x.ptr
}

//gopherjs:replace
func (v Value) Elem() Value {
	switch k := v.kind(); k {
	case abi.Interface:
		val := v.object()
		if val == js.Global.Get("$ifaceNil") {
			return Value{}
		}
		typ := abi.ReflectType(val.Get("constructor"))
		return makeValue(typ, val.Get("$val"), v.flag.ro())

	case abi.Pointer:
		if v.IsNil() {
			return Value{}
		}
		val := v.object()
		tt := (*abi.PtrType)(unsafe.Pointer(v.typ))
		fl := v.flag&flagRO | flagIndir | flagAddr
		fl |= flag(tt.Elem.Kind())
		return Value{tt.Elem, unsafe.Pointer(abi.WrapJsObject(tt.Elem, val).Unsafe()), fl}

	default:
		panic(&ValueError{"reflect.Value.Elem", k})
	}
}

//gopherjs:purge Unused type
type emptyInterface struct{}

//gopherjs:purge Unused method for emptyInterface
func unpackEface(i any) Value

//gopherjs:purge Unused method for emptyInterface
func packEface(v Value) any
