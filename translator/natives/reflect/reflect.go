// +build js

package reflect

import (
	"github.com/gopherjs/gopherjs/js"
	"unsafe"
)

// temporary
func init() {
	a := false
	if a {
		isWrapped(nil)
		copyStruct(nil, nil, nil)
		makeIndir(nil, nil)
		jsObject()
	}
}

func jsType(typ Type) js.Object {
	return js.InternalObject(typ).Get("jsType")
}

func reflectType(typ js.Object) *rtype {
	return typ.Call("reflectType").Interface().(*rtype)
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

func makeIndir(t Type, v js.Object) iword {
	rt := t.(*rtype)
	if rt.size > 4 {
		return iword(js.Global.Call("go$newDataPointer", v, jsType(rt.ptrTo())).Unsafe())
	}
	return iword(v.Unsafe())
}

func makeIndirValue(t Type, v js.Object) Value {
	rt := t.(*rtype)
	return Value{rt, unsafe.Pointer(js.Global.Call("go$newDataPointer", v, jsType(rt.ptrTo())).Unsafe()), flagIndir | flag(t.Kind())<<flagKindShift}
}

func MakeSlice(typ Type, len, cap int) Value {
	if typ.Kind() != Slice {
		panic("reflect.MakeSlice of non-slice type")
	}
	if len < 0 {
		panic("reflect.MakeSlice: negative len")
	}
	if cap < 0 {
		panic("reflect.MakeSlice: negative cap")
	}
	if len > cap {
		panic("reflect.MakeSlice: len > cap")
	}

	return makeIndirValue(typ, jsType(typ).Call("make", len, cap, func() js.Object { return zeroVal(typ.Elem()) }))
}

func jsObject() *rtype {
	return reflectType(js.Global.Get("go$packages").Get("github.com/gopherjs/gopherjs/js").Get("Object"))
}

func TypeOf(i interface{}) Type {
	if i == nil {
		return nil
	}
	c := js.InternalObject(i).Get("constructor")
	if c.Get("kind").IsUndefined() { // js.Object
		return jsObject()
	}
	return reflectType(c)
}

func ValueOf(i interface{}) Value {
	if i == nil {
		return Value{}
	}
	c := js.InternalObject(i).Get("constructor")
	if c.Get("kind").IsUndefined() { // js.Object
		return Value{jsObject(), unsafe.Pointer(js.InternalObject(i).Unsafe()), flag(Interface) << flagKindShift}
	}
	typ := reflectType(c)
	if typ.Kind() != Array && typ.Kind() != Struct && typ.Size() > ptrSize {
		return makeIndirValue(typ, js.InternalObject(i).Get("go$val"))
	}
	return Value{typ, unsafe.Pointer(js.InternalObject(i).Get("go$val").Unsafe()), flag(typ.Kind()) << flagKindShift}
}

func arrayOf(count int, elem Type) Type {
	return reflectType(js.Global.Call("go$arrayType", jsType(elem), count))
}

func ChanOf(dir ChanDir, t Type) Type {
	return reflectType(js.Global.Call("go$chanType", jsType(t), dir == SendDir, dir == RecvDir))
}

func MapOf(key, elem Type) Type {
	switch key.Kind() {
	case Func, Map, Slice:
		panic("reflect.MapOf: invalid key type " + key.String())
	}

	return reflectType(js.Global.Call("go$mapType", jsType(key), jsType(elem)))
}

func (t *rtype) ptrTo() *rtype {
	return reflectType(js.Global.Call("go$ptrType", jsType(t)))
}

func SliceOf(t Type) Type {
	return reflectType(js.Global.Call("go$sliceType", jsType(t)))
}

func Zero(typ Type) Value {
	return Value{typ.(*rtype), unsafe.Pointer(zeroVal(typ).Unsafe()), flag(typ.Kind()) << flagKindShift}
}

func unsafe_New(typ *rtype) unsafe.Pointer {
	switch typ.Kind() {
	case Struct:
		return unsafe.Pointer(jsType(typ).Get("Ptr").New().Unsafe())
	case Array:
		return unsafe.Pointer(zeroVal(typ).Unsafe())
	default:
		return unsafe.Pointer(js.Global.Call("go$newDataPointer", zeroVal(typ), jsType(typ.ptrTo())).Unsafe())
	}
}

func makechan(typ *rtype, size uint64) (ch iword) {
	return iword(jsType(typ).New().Unsafe())
}

func chancap(ch iword) int {
	js.Global.Call("go$notSupported", "channels")
	panic("unreachable")
}

func chanclose(ch iword) {
	js.Global.Call("go$notSupported", "channels")
	panic("unreachable")
}

func chanlen(ch iword) int {
	js.Global.Call("go$notSupported", "channels")
	panic("unreachable")
}

func chanrecv(t *rtype, ch iword, nb bool) (val iword, selected, received bool) {
	js.Global.Call("go$notSupported", "channels")
	panic("unreachable")
}

func chansend(t *rtype, ch iword, val iword, nb bool) bool {
	js.Global.Call("go$notSupported", "channels")
	panic("unreachable")
}

func makemap(t *rtype) (m iword) {
	return iword(js.Global.Get("Go$Map").New().Unsafe())
}

func mapaccess(t *rtype, m iword, key iword) (val iword, ok bool) {
	k := js.InternalObject(key)
	if !k.Get("go$key").IsUndefined() {
		k = k.Call("go$key")
	}
	entry := js.InternalObject(m).Get(k.Str())
	if entry.IsUndefined() {
		return nil, false
	}
	return makeIndir(t.Elem(), entry.Get("v")), true
}

func mapassign(t *rtype, m iword, key, val iword, ok bool) {
	k := js.InternalObject(key)
	if !k.Get("go$key").IsUndefined() {
		k = k.Call("go$key")
	}
	if !ok {
		js.InternalObject(m).Delete(k.Str())
		return
	}
	jsVal := js.InternalObject(val)
	if t.Elem().Kind() == Struct {
		newVal := js.Global.Get("Object").New()
		copyStruct(newVal, jsVal, t.Elem())
		jsVal = newVal
	}
	entry := js.Global.Get("Object").New()
	entry.Set("k", key)
	entry.Set("v", jsVal)
	js.InternalObject(m).Set(k.Str(), entry)
}

type mapIter struct {
	t    Type
	m    js.Object
	keys js.Object
	i    int
}

func mapiterinit(t *rtype, m iword) *byte {
	return (*byte)(unsafe.Pointer(&mapIter{t, js.InternalObject(m), js.Global.Call("go$keys", m), 0}))
}

func mapiterkey(it *byte) (key iword, ok bool) {
	iter := js.InternalObject(it)
	k := iter.Get("keys").Index(iter.Get("i").Int())
	return makeIndir(iter.Get("t").Interface().(*rtype).Key(), iter.Get("m").Get(k.Str()).Get("k")), true
	// k := iter.keys.Index(iter.i)
	// return makeIndir(iter.t.Key(), iter.m.G
}

func mapiternext(it *byte) {
	iter := js.InternalObject(it)
	iter.Set("i", iter.Get("i").Int()+1)
}

func maplen(m iword) int {
	return js.Global.Call("go$keys", m).Length()
}

func (v Value) iword() iword {
	if v.flag&flagIndir != 0 && v.typ.Kind() != Array && v.typ.Kind() != Struct {
		return iword(js.InternalObject(v.val).Call("go$get").Unsafe())
	}
	return iword(v.val)
}

func (v Value) Cap() int {
	k := v.kind()
	switch k {
	case Array:
		return v.typ.Len()
	// case Chan:
	// 	return int(chancap(v.iword()))
	case Slice:
		return js.InternalObject(v.iword()).Get("capacity").Int()
	}
	panic(&ValueError{"reflect.Value.Cap", k})
}

func (v Value) Len() int {
	k := v.kind()
	switch k {
	case Array, Slice, String:
		return js.InternalObject(v.iword()).Length()
	// case Chan:
	// 	return chanlen(v.iword())
	case Map:
		return js.Global.Call("go$keys", v.iword()).Length()
	}
	panic(&ValueError{"reflect.Value.Len", k})
}
