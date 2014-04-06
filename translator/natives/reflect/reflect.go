// +build js

package reflect

import (
	"github.com/gopherjs/gopherjs/js"
)

// temporary
func init() {
	a := false
	if a {
		isWrapped(nil)
		copyStruct(nil, nil, nil)
		zeroVal(nil)
		makeIndir(nil, nil)
		jsObject()
	}
}

func jsType(typ Type) js.Object {
	return js.InternalObject(typ).Get("jsType")
}

func isWrapped(typ Type) bool {
	switch typ.Kind() {
	case Bool, Int, Int8, Int16, Int32, Uint, Uint8, Uint16, Uint32, Uintptr, Float32, Float64, Array, Map, Func, String, Struct:
		return true
	case Ptr:
		return typ.Elem().Kind() == Array
	}
	return false
}

func copyStruct(dst, src js.Object, typ Type) {
	fields := jsType(typ).Get("fields")
	for i := 0; i < fields.Length(); i++ {
		name := fields.Index(i).Index(0).Str()
		dst.Set(name, src.Get(name))
	}
}

func zeroVal(typ Type) js.Object {
	switch typ.Kind() {
	case Bool:
		return js.InternalObject(false)
	case Int, Int8, Int16, Int32, Uint, Uint8, Uint16, Uint32, Uintptr, Float32, Float64:
		return js.InternalObject(0)
	case Int64, Uint64, Complex64, Complex128:
		return jsType(typ).New(0, 0)
	case Array:
		elemType := typ.Elem()
		return js.Global.Call("go$makeNativeArray", jsType(elemType).Get("kind"), typ.Len(), func() js.Object { return zeroVal(elemType) })
	case Func:
		return js.Global.Get("go$throwNilPointerError")
	case Interface:
		return nil
	case Map:
		return js.InternalObject(false)
	case Chan, Ptr, Slice:
		return jsType(typ).Get("nil")
	case String:
		return js.InternalObject("")
	case Struct:
		return jsType(typ).Get("Ptr").New()
	default:
		panic(&ValueError{"reflect.Zero", typ.Kind()})
	}
}

func makeIndir(t *rtype, v js.Object) js.Object {
	if t.size > 4 {
		return js.Global.Call("go$newDataPointer", v, jsType(t.ptrTo()))
	}
	return v
}

func jsObject() *rtype {
	return js.Global.Get("go$packages").Get("github.com/gopherjs/gopherjs/js").Get("Object").Call("reflectType").Interface().(*rtype)
}

func TypeOf(i interface{}) Type {
	if i == nil {
		return nil
	}
	if js.InternalObject(i).Get("constructor").Get("kind").IsUndefined() { // js.Object
		return jsObject()
	}
	return js.InternalObject(i).Get("constructor").Call("reflectType").Interface().(*rtype)
}
