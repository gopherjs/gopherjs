//go:build js

package reflect

import (
	"unsafe"

	"internal/abi"
	"internal/itoa"

	"github.com/gopherjs/gopherjs/js"
)

//gopherjs:replace
func cvtDirect(v Value, typ Type) Value {
	srcVal := v.object()
	if srcVal == v.typ().JsType().Get("nil") {
		return makeValue(toAbiType(typ), jsType(typ).Get("nil"), v.flag)
	}

	var val *js.Object
	switch k := typ.Kind(); k {
	case Slice:
		slice := jsType(typ).New(srcVal.Get("$array"))
		slice.Set("$offset", srcVal.Get("$offset"))
		slice.Set("$length", srcVal.Get("$length"))
		slice.Set("$capacity", srcVal.Get("$capacity"))
		val = js.Global.Call("$newDataPointer", slice, jsType(PtrTo(typ)))
	case Ptr:
		switch typ.Elem().Kind() {
		case Struct:
			if toAbiType(typ.Elem()) == v.typ().Elem() {
				val = srcVal
				break
			}
			val = jsType(typ).New()
			abi.CopyStruct(val, srcVal, toAbiType(typ.Elem()))
		case Array:
			// Unlike other pointers, array pointers are "wrapped" types (see
			// isWrapped() in the compiler package), and are represented by a native
			// javascript array object here.
			val = srcVal
		default:
			val = jsType(typ).New(srcVal.Get("$get"), srcVal.Get("$set"))
		}
	case Struct:
		val = jsType(typ).Get("ptr").New()
		abi.CopyStruct(val, srcVal, toAbiType(typ))
	case Array, Bool, Chan, Func, Interface, Map, String, UnsafePointer:
		val = js.InternalObject(v.ptr)
	default:
		panic(&ValueError{Method: "reflect.Convert", Kind: k})
	}
	return Value{
		typ_: typ.common(),
		ptr:  unsafe.Pointer(val.Unsafe()),
		flag: v.flag.ro() | v.flag&flagIndir | flag(typ.Kind()),
	}
}

// convertOp: []T -> *[N]T
//
//gopherjs:replace
func cvtSliceArrayPtr(v Value, t Type) Value {
	slice := v.object()

	slen := slice.Get("$length").Int()
	alen := t.Elem().Len()
	if alen > slen {
		panic("reflect: cannot convert slice with length " + itoa.Itoa(slen) + " to pointer to array with length " + itoa.Itoa(alen))
	}
	array := js.Global.Call("$sliceToGoArray", slice, jsType(t))
	return Value{
		typ_: t.common(),
		ptr:  unsafe.Pointer(array.Unsafe()),
		flag: v.flag&^(flagIndir|flagAddr|flagKindMask) | flag(Ptr),
	}
}

// convertOp: []T -> [N]T
//
//gopherjs:replace
func cvtSliceArray(v Value, t Type) Value {
	n := t.Len()
	if n > v.Len() {
		panic("reflect: cannot convert slice with length " + itoa.Itoa(v.Len()) + " to array with length " + itoa.Itoa(n))
	}

	slice := v.object()
	dst := MakeSlice(SliceOf(t.Elem()), n, n).object()
	js.Global.Call("$copySlice", dst, slice)

	arr := dst.Get("$array")
	return Value{
		typ_: t.common(),
		ptr:  unsafe.Pointer(arr.Unsafe()),
		flag: v.flag&^(flagAddr|flagKindMask) | flag(Array),
	}
}

//gopherjs:replace
func Copy(dst, src Value) int {
	dk := dst.kind()
	if dk != Array && dk != Slice {
		panic(&ValueError{Method: "reflect.Copy", Kind: dk})
	}
	if dk == Array {
		dst.mustBeAssignable()
	}
	dst.mustBeExported()

	sk := src.kind()
	var stringCopy bool
	if sk != Array && sk != Slice {
		stringCopy = sk == String && dst.typ().Elem().Kind() == abi.Uint8
		if !stringCopy {
			panic(&ValueError{Method: "reflect.Copy", Kind: sk})
		}
	}
	src.mustBeExported()

	if !stringCopy {
		typesMustMatch("reflect.Copy", toRType(dst.typ().Elem()), toRType(src.typ().Elem()))
	}

	dstVal := dst.object()
	if dk == Array {
		dstVal = jsType(SliceOf(toRType(dst.typ().Elem()))).New(dstVal)
	}

	srcVal := src.object()
	if sk == Array {
		srcVal = jsType(SliceOf(toRType(src.typ().Elem()))).New(srcVal)
	}

	if stringCopy {
		return js.Global.Call("$copyString", dstVal, srcVal).Int()
	}
	return js.Global.Call("$copySlice", dstVal, srcVal).Int()
}

//gopherjs:replace
func valueInterface(v Value, safe bool) any {
	if v.flag == 0 {
		panic(&ValueError{Method: "reflect.Value.Interface", Kind: 0})
	}
	if safe && v.flag&flagRO != 0 {
		panic("reflect.Value.Interface: cannot return value obtained from unexported field or method")
	}
	if v.flag&flagMethod != 0 {
		v = makeMethodValue("Interface", v)
	}

	if v.typ().IsWrapped() {
		jsTyp := v.typ().JsType()
		if v.flag&flagIndir != 0 && v.Kind() == Struct {
			cv := jsTyp.Call("zero")
			abi.CopyStruct(cv, v.object(), v.typ())
			return any(unsafe.Pointer(jsTyp.New(cv).Unsafe()))
		}
		return any(unsafe.Pointer(jsTyp.New(v.object()).Unsafe()))
	}
	return any(unsafe.Pointer(v.object().Unsafe()))
}

//gopherjs:new
func (t *rtype) pointers() bool {
	switch t.Kind() {
	case Ptr, Map, Chan, Func, Struct, Array:
		return true
	default:
		return false
	}
}

//gopherjs:replace
func (t *rtype) Comparable() bool {
	return toAbiType(t).Comparable()
}

//gopherjs:replace Used pointer cast to interface kind type.
func (t *rtype) NumMethod() int {
	if tt := t.common().InterfaceType(); tt != nil {
		return tt.NumMethod()
	}
	return len(t.exportedMethods())
}

//gopherjs:replace
func (t *rtype) Method(i int) (m Method) {
	if t.Kind() == Interface {
		tt := toInterfaceType(t.common())
		return tt.Method(i)
	}
	methods := t.exportedMethods()
	if i < 0 || i >= len(methods) {
		panic("reflect: Method index out of range")
	}
	p := methods[i]
	pname := t.nameOff(p.Name)
	m.Name = pname.Name()
	fl := flag(Func)
	mtyp := t.typeOff(p.Mtyp)
	ft := mtyp.FuncType()
	in := make([]Type, 0, 1+ft.NumIn())
	in = append(in, t)
	for _, arg := range ft.InSlice() {
		in = append(in, toRType(arg))
	}
	out := make([]Type, 0, ft.NumOut())
	for _, ret := range ft.OutSlice() {
		out = append(out, toRType(ret))
	}
	mt := FuncOf(in, out, ft.IsVariadic())
	m.Type = mt
	prop := js.Global.Call("$methodSet", js.InternalObject(t).Get("jsType")).Index(i).Get("prop").String()
	fn := js.MakeFunc(func(this *js.Object, arguments []*js.Object) any {
		rcvr := arguments[0]
		return rcvr.Get(prop).Call("apply", rcvr, arguments[1:])
	})
	m.Func = Value{toAbiType(mt), unsafe.Pointer(fn.Unsafe()), fl}

	m.Index = i
	return m
}

//gopherjs:new
var selectHelper = js.Global.Get("$select").Interface().(func(...any) *js.Object)

//gopherjs:replace
func chanrecv(ch unsafe.Pointer, nb bool, val unsafe.Pointer) (selected, received bool) {
	comms := [][]*js.Object{{js.InternalObject(ch)}}
	if nb {
		comms = append(comms, []*js.Object{})
	}
	selectRes := selectHelper(comms)
	if nb && selectRes.Index(0).Int() == 1 {
		return false, false
	}
	recvRes := selectRes.Index(1)
	js.InternalObject(val).Call("$set", recvRes.Index(0))
	return true, recvRes.Index(1).Bool()
}

//gopherjs:replace
func chansend(ch unsafe.Pointer, val unsafe.Pointer, nb bool) bool {
	comms := [][]*js.Object{{js.InternalObject(ch), js.InternalObject(val).Call("$get")}}
	if nb {
		comms = append(comms, []*js.Object{})
	}
	selectRes := selectHelper(comms)
	if nb && selectRes.Index(0).Int() == 1 {
		return false
	}
	return true
}

//gopherjs:replace
func rselect(rselects []runtimeSelect) (chosen int, recvOK bool) {
	comms := make([][]*js.Object, len(rselects))
	for i, s := range rselects {
		switch SelectDir(s.dir) {
		case SelectDefault:
			comms[i] = []*js.Object{}
		case SelectRecv:
			ch := js.Global.Get("$chanNil")
			if js.InternalObject(s.ch) != js.InternalObject(0) {
				ch = js.InternalObject(s.ch)
			}
			comms[i] = []*js.Object{ch}
		case SelectSend:
			ch := js.Global.Get("$chanNil")
			var val *js.Object
			if js.InternalObject(s.ch) != js.InternalObject(0) {
				ch = js.InternalObject(s.ch)
				val = js.InternalObject(s.val).Call("$get")
			}
			comms[i] = []*js.Object{ch, val}
		}
	}
	selectRes := selectHelper(comms)
	c := selectRes.Index(0).Int()
	if SelectDir(rselects[c].dir) == SelectRecv {
		recvRes := selectRes.Index(1)
		js.InternalObject(rselects[c].val).Call("$set", recvRes.Index(0))
		return c, recvRes.Index(1).Bool()
	}
	return c, false
}

//gopherjs:replace
func DeepEqual(a1, a2 any) bool {
	i1 := js.InternalObject(a1)
	i2 := js.InternalObject(a2)
	if i1 == i2 {
		return true
	}
	if i1 == nil || i2 == nil || i1.Get("constructor") != i2.Get("constructor") {
		return false
	}
	return deepValueEqualJs(ValueOf(a1), ValueOf(a2), nil)
}

//gopherjs:new
func deepValueEqualJs(v1, v2 Value, visited [][2]unsafe.Pointer) bool {
	if !v1.IsValid() || !v2.IsValid() {
		return !v1.IsValid() && !v2.IsValid()
	}
	if v1.Type() != v2.Type() {
		return false
	}
	if v1.typ() == abi.JsObjectPtr {
		return abi.UnwrapJsObject(abi.JsObjectPtr, v1.object()) == abi.UnwrapJsObject(abi.JsObjectPtr, v2.object())
	}

	switch v1.Kind() {
	case Array, Map, Slice, Struct:
		for _, entry := range visited {
			if v1.ptr == entry[0] && v2.ptr == entry[1] {
				return true
			}
		}
		visited = append(visited, [2]unsafe.Pointer{v1.ptr, v2.ptr})
	}

	switch v1.Kind() {
	case Array, Slice:
		if v1.Kind() == Slice {
			if v1.IsNil() != v2.IsNil() {
				return false
			}
			if v1.object() == v2.object() {
				return true
			}
		}
		n := v1.Len()
		if n != v2.Len() {
			return false
		}
		for i := 0; i < n; i++ {
			if !deepValueEqualJs(v1.Index(i), v2.Index(i), visited) {
				return false
			}
		}
		return true
	case Interface:
		if v1.IsNil() || v2.IsNil() {
			return v1.IsNil() && v2.IsNil()
		}
		return deepValueEqualJs(v1.Elem(), v2.Elem(), visited)
	case Ptr:
		return deepValueEqualJs(v1.Elem(), v2.Elem(), visited)
	case Struct:
		n := v1.NumField()
		for i := 0; i < n; i++ {
			if !deepValueEqualJs(v1.Field(i), v2.Field(i), visited) {
				return false
			}
		}
		return true
	case Map:
		if v1.IsNil() != v2.IsNil() {
			return false
		}
		if v1.object() == v2.object() {
			return true
		}
		keys := v1.MapKeys()
		if len(keys) != v2.Len() {
			return false
		}
		for _, k := range keys {
			val1 := v1.MapIndex(k)
			val2 := v2.MapIndex(k)
			if !val1.IsValid() || !val2.IsValid() || !deepValueEqualJs(val1, val2, visited) {
				return false
			}
		}
		return true
	case Func:
		return v1.IsNil() && v2.IsNil()
	case UnsafePointer:
		return v1.object() == v2.object()
	}

	return js.Global.Call("$interfaceIsEqual", js.InternalObject(valueInterface(v1, false)), js.InternalObject(valueInterface(v2, false))).Bool()
}

//gopherjs:new
func stringsLastIndex(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

//gopherjs:new
func stringsHasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

//gopherjs:replace
func verifyNotInHeapPtr(p uintptr) bool {
	// Go runtime uses this method to make sure that a uintptr won't crash GC if
	// interpreted as a heap pointer. This is not relevant for GopherJS, so we can
	// always return true.
	return true
}
