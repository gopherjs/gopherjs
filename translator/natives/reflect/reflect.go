// +build js

package reflect

import (
	"github.com/gopherjs/gopherjs/js"
	"unsafe"
)

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

func makeIword(t Type, v js.Object) iword {
	if t.Size() > ptrSize && t.Kind() != Array && t.Kind() != Struct {
		return iword(js.Global.Call("go$newDataPointer", v, jsType(PtrTo(t))).Unsafe())
	}
	return iword(v.Unsafe())
}

func makeValue(t Type, v js.Object, fl flag) Value {
	rt := t.common()
	if t.Size() > ptrSize && t.Kind() != Array && t.Kind() != Struct {
		return Value{rt, unsafe.Pointer(js.Global.Call("go$newDataPointer", v, jsType(rt.ptrTo())).Unsafe()), fl | flag(t.Kind())<<flagKindShift | flagIndir}
	}
	return Value{rt, unsafe.Pointer(v.Unsafe()), fl | flag(t.Kind())<<flagKindShift}
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

	return makeValue(typ, jsType(typ).Call("make", len, cap, func() js.Object { return zeroVal(typ.Elem()) }), 0)
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
	return makeValue(reflectType(c), js.InternalObject(i).Get("go$val"), 0)
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
	return Value{typ.common(), unsafe.Pointer(zeroVal(typ).Unsafe()), flag(typ.Kind()) << flagKindShift}
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

func makeInt(f flag, bits uint64, t Type) Value {
	typ := t.common()
	if typ.size > ptrSize {
		// Assume ptrSize >= 4, so this must be uint64.
		ptr := unsafe_New(typ)
		*(*uint64)(unsafe.Pointer(ptr)) = bits
		return Value{typ, ptr, f | flagIndir | flag(typ.Kind())<<flagKindShift}
	}
	var w iword
	switch typ.Kind() {
	case Int8:
		*(*int8)(unsafe.Pointer(&w)) = int8(bits)
	case Int16:
		*(*int16)(unsafe.Pointer(&w)) = int16(bits)
	case Int, Int32:
		*(*int32)(unsafe.Pointer(&w)) = int32(bits)
	case Uint8:
		*(*uint8)(unsafe.Pointer(&w)) = uint8(bits)
	case Uint16:
		*(*uint16)(unsafe.Pointer(&w)) = uint16(bits)
	case Uint, Uint32, Uintptr:
		*(*uint32)(unsafe.Pointer(&w)) = uint32(bits)
	}
	return Value{typ, unsafe.Pointer(w), f | flag(typ.Kind())<<flagKindShift}
}

func MakeFunc(typ Type, fn func(args []Value) (results []Value)) Value {
	if typ.Kind() != Func {
		panic("reflect: call of MakeFunc with non-Func type")
	}

	t := typ.common()
	ftyp := (*funcType)(unsafe.Pointer(t))

	fv := func() js.Object {
		args := make([]Value, ftyp.NumIn())
		for i := range args {
			argType := ftyp.In(i).common()
			args[i] = Value{argType, unsafe.Pointer(js.Arguments[i].Unsafe()), flag(argType.Kind()) << flagKindShift}
		}
		resultsSlice := fn(args)
		switch ftyp.NumOut() {
		case 0:
			return nil
		case 1:
			return js.InternalObject(resultsSlice[0].iword())
		default:
			results := js.Global.Get("Array").New(ftyp.NumOut())
			for i, r := range resultsSlice {
				results.SetIndex(i, js.InternalObject(r.iword()))
			}
			return results
		}
	}

	return Value{t, unsafe.Pointer(js.InternalObject(fv).Unsafe()), flag(Func) << flagKindShift}
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
	return makeIword(t.Elem(), entry.Get("v")), true
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
	return makeIword(iter.Get("t").Interface().(*rtype).Key(), iter.Get("m").Get(k.Str()).Get("k")), true
	// k := iter.keys.Index(iter.i)
	// return makeIword(iter.t.Key(), iter.m.G
}

func mapiternext(it *byte) {
	iter := js.InternalObject(it)
	iter.Set("i", iter.Get("i").Int()+1)
}

func maplen(m iword) int {
	return js.Global.Call("go$keys", m).Length()
}

func cvtDirect(v Value, typ Type) Value {
	var srcVal = js.InternalObject(v.iword())
	if srcVal == jsType(v.typ).Get("nil") {
		return makeValue(typ, jsType(typ).Get("nil"), v.flag)
	}

	var val js.Object
	switch k := typ.Kind(); k {
	case Chan:
		val = jsType(typ).New()
	case Slice:
		slice := jsType(typ).New(srcVal.Get("array"))
		slice.Set("offset", srcVal.Get("offset"))
		slice.Set("length", srcVal.Get("length"))
		slice.Set("capacity", srcVal.Get("capacity"))
		val = js.Global.Call("go$newDataPointer", slice, jsType(PtrTo(typ)))
	case Ptr:
		if typ.Elem().Kind() == Struct {
			if typ.Elem() == v.typ.Elem() {
				val = srcVal
				break
			}
			val = jsType(typ).New()
			copyStruct(val, srcVal, typ.Elem())
			break
		}
		val = jsType(typ).New(srcVal.Get("go$get"), srcVal.Get("go$set"))
	case Struct:
		val = jsType(typ).Get("Ptr").New()
		copyStruct(val, srcVal, typ)
	case Array, Func, Interface, Map, String:
		val = js.InternalObject(v.val)
	default:
		panic(&ValueError{"reflect.Convert", k})
	}
	return Value{typ.common(), unsafe.Pointer(val.Unsafe()), v.flag&(flagRO|flagIndir) | flag(typ.Kind())<<flagKindShift}
}

func Copy(dst, src Value) int {
	dk := dst.kind()
	if dk != Array && dk != Slice {
		panic(&ValueError{"reflect.Copy", dk})
	}
	if dk == Array {
		dst.mustBeAssignable()
	}
	dst.mustBeExported()

	sk := src.kind()
	if sk != Array && sk != Slice {
		panic(&ValueError{"reflect.Copy", sk})
	}
	src.mustBeExported()

	typesMustMatch("reflect.Copy", dst.typ.Elem(), src.typ.Elem())

	dstVal := js.InternalObject(dst.iword())
	if dk == Array {
		dstVal = jsType(SliceOf(dst.typ.Elem())).New(dstVal)
	}

	srcVal := js.InternalObject(src.iword())
	if sk == Array {
		srcVal = jsType(SliceOf(src.typ.Elem())).New(srcVal)
	}

	return js.Global.Call("go$copySlice", dstVal, srcVal).Int()
}

func methodReceiver(op string, v Value, i int) (*rtype, unsafe.Pointer, iword) {
	var t *rtype
	var name string
	if v.typ.Kind() == Interface {
		tt := (*interfaceType)(unsafe.Pointer(v.typ))
		if i < 0 || i >= len(tt.methods) {
			panic("reflect: internal error: invalid method index")
		}
		if v.IsNil() {
			panic("reflect: " + op + " of method on nil interface value")
		}
		m := &tt.methods[i]
		if m.pkgPath != nil {
			panic("reflect: " + op + " of unexported method")
		}
		t = m.typ
		name = *m.name
	} else {
		ut := v.typ.uncommon()
		if ut == nil || i < 0 || i >= len(ut.methods) {
			panic("reflect: internal error: invalid method index")
		}
		m := &ut.methods[i]
		if m.pkgPath != nil {
			panic("reflect: " + op + " of unexported method")
		}
		t = m.mtyp
		name = jsType(v.typ).Get("methods").Index(i).Index(0).Str()
	}
	rcvr := js.InternalObject(v.iword())
	if isWrapped(v.typ) {
		rcvr = jsType(v.typ).New(rcvr)
	}
	return t, unsafe.Pointer(rcvr.Get(name).Unsafe()), iword(rcvr.Unsafe())
}

func valueInterface(v Value, safe bool) interface{} {
	if v.flag == 0 {
		panic(&ValueError{"reflect.Value.Interface", 0})
	}
	if safe && v.flag&flagRO != 0 {
		panic("reflect.Value.Interface: cannot return value obtained from unexported field or method")
	}
	if v.flag&flagMethod != 0 {
		v = makeMethodValue("Interface", v)
	}

	if isWrapped(v.typ) {
		return interface{}(unsafe.Pointer(jsType(v.typ).New(js.InternalObject(v.iword())).Unsafe()))
	}
	return interface{}(unsafe.Pointer(js.InternalObject(v.iword()).Unsafe()))
}

func ifaceE2I(t *rtype, src interface{}, dst unsafe.Pointer) {
	js.InternalObject(dst).Call("go$set", js.InternalObject(src))
}

func methodName() string {
	return "?FIXME?"
}

func makeMethodValue(op string, v Value) Value {
	if v.flag&flagMethod == 0 {
		panic("reflect: internal error: invalid use of makePartialFunc")
	}

	_, fn, rcvr := methodReceiver(op, v, int(v.flag)>>flagMethodShift)
	fv := func() js.Object {
		return js.InternalObject(fn).Call("apply", rcvr, js.Arguments)
	}
	return Value{v.Type().common(), unsafe.Pointer(js.InternalObject(fv).Unsafe()), v.flag&flagRO | flag(Func)<<flagKindShift}
}

func (t *uncommonType) Method(i int) (m Method) {
	if t == nil || i < 0 || i >= len(t.methods) {
		panic("reflect: Method index out of range")
	}
	p := &t.methods[i]
	if p.name != nil {
		m.Name = *p.name
	}
	fl := flag(Func) << flagKindShift
	if p.pkgPath != nil {
		m.PkgPath = *p.pkgPath
		fl |= flagRO
	}
	mt := p.typ
	m.Type = mt
	name := js.InternalObject(t).Get("jsType").Get("methods").Index(i).Index(0).Str()
	fn := func(rcvr js.Object) js.Object {
		return rcvr.Get(name).Call("apply", rcvr, js.Arguments[1:])
	}
	m.Func = Value{mt, unsafe.Pointer(js.InternalObject(fn).Unsafe()), fl}
	m.Index = i
	return
}

func (v Value) iword() iword {
	if v.flag&flagIndir != 0 && v.typ.Kind() != Array && v.typ.Kind() != Struct {
		val := js.InternalObject(v.val).Call("go$get")
		if !val.IsNull() && val.Get("constructor") != jsType(v.typ) {
			switch v.typ.Kind() {
			case Uint64, Int64:
				val = jsType(v.typ).New(val.Get("high"), val.Get("low"))
			case Complex64, Complex128:
				val = jsType(v.typ).New(val.Get("real"), val.Get("imag"))
			case Slice:
				if val == val.Get("constructor").Get("nil") {
					val = jsType(v.typ).Get("nil")
					break
				}
				newVal := jsType(v.typ).New(val.Get("array"))
				newVal.Set("offset", val.Get("offset"))
				newVal.Set("length", val.Get("length"))
				newVal.Set("capacity", val.Get("capacity"))
				val = newVal
			}
		}
		return iword(val.Unsafe())
	}
	return iword(v.val)
}

func (v Value) call(op string, in []Value) []Value {
	t := v.typ
	var (
		fn   unsafe.Pointer
		rcvr iword
	)
	if v.flag&flagMethod != 0 {
		t, fn, rcvr = methodReceiver(op, v, int(v.flag)>>flagMethodShift)
	} else {
		fn = unsafe.Pointer(v.iword())
	}

	if fn == nil {
		panic("reflect.Value.Call: call of nil function")
	}

	isSlice := op == "CallSlice"
	n := t.NumIn()
	if isSlice {
		if !t.IsVariadic() {
			panic("reflect: CallSlice of non-variadic function")
		}
		if len(in) < n {
			panic("reflect: CallSlice with too few input arguments")
		}
		if len(in) > n {
			panic("reflect: CallSlice with too many input arguments")
		}
	} else {
		if t.IsVariadic() {
			n--
		}
		if len(in) < n {
			panic("reflect: Call with too few input arguments")
		}
		if !t.IsVariadic() && len(in) > n {
			panic("reflect: Call with too many input arguments")
		}
	}
	for _, x := range in {
		if x.Kind() == Invalid {
			panic("reflect: " + op + " using zero Value argument")
		}
	}
	for i := 0; i < n; i++ {
		if xt, targ := in[i].Type(), t.In(i); !xt.AssignableTo(targ) {
			panic("reflect: " + op + " using " + xt.String() + " as type " + targ.String())
		}
	}
	if !isSlice && t.IsVariadic() {
		// prepare slice for remaining values
		m := len(in) - n
		slice := MakeSlice(t.In(n), m, m)
		elem := t.In(n).Elem()
		for i := 0; i < m; i++ {
			x := in[n+i]
			if xt := x.Type(); !xt.AssignableTo(elem) {
				panic("reflect: cannot use " + xt.String() + " as type " + elem.String() + " in " + op)
			}
			slice.Index(i).Set(x)
		}
		origIn := in
		in = make([]Value, n+1)
		copy(in[:n], origIn)
		in[n] = slice
	}

	nin := len(in)
	if nin != t.NumIn() {
		panic("reflect.Value.Call: wrong argument count")
	}
	nout := t.NumOut()

	argsArray := js.Global.Get("Array").New(t.NumIn())
	for i, arg := range in {
		argsArray.SetIndex(i, arg.assignTo("reflect.Value.Call", t.In(i).common(), nil).iword())
	}
	results := js.InternalObject(fn).Call("apply", rcvr, argsArray)

	switch nout {
	case 0:
		return nil
	case 1:
		return []Value{makeValue(t.Out(0), results, 0)}
	default:
		ret := make([]Value, nout)
		for i := range ret {
			ret[i] = makeValue(t.Out(i), results.Index(i), 0)
		}
		return ret
	}
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

func (v Value) Elem() Value {
	switch k := v.kind(); k {
	case Interface:
		val := js.InternalObject(v.iword())
		if val.IsNull() {
			return Value{}
		}
		typ := reflectType(val.Get("constructor"))
		return makeValue(typ, val.Get("go$val"), v.flag&flagRO)

	case Ptr:
		if v.IsNil() {
			return Value{}
		}
		val := v.iword()
		tt := (*ptrType)(unsafe.Pointer(v.typ))
		fl := v.flag&flagRO | flagIndir | flagAddr
		fl |= flag(tt.elem.Kind()) << flagKindShift
		return Value{tt.elem, unsafe.Pointer(val), fl}

	default:
		panic(&ValueError{"reflect.Value.Elem", k})
	}
}

func (v Value) Field(i int) Value {
	v.mustBe(Struct)
	tt := (*structType)(unsafe.Pointer(v.typ))
	if i < 0 || i >= len(tt.fields) {
		panic("reflect: Field index out of range")
	}

	field := &tt.fields[i]
	name := jsType(v.typ).Get("fields").Index(i).Index(0).Str()
	typ := field.typ

	fl := v.flag & (flagRO | flagIndir | flagAddr)
	if field.pkgPath != nil {
		fl |= flagRO
	}
	fl |= flag(typ.Kind()) << flagKindShift

	s := js.InternalObject(v.val)
	if fl&flagIndir != 0 && typ.Kind() != Array && typ.Kind() != Struct {
		return Value{typ, unsafe.Pointer(jsType(PtrTo(typ)).New(func() js.Object { return s.Get(name) }, func(v js.Object) { s.Set(name, v) }).Unsafe()), fl}
	}
	return makeValue(typ, s.Get(name), fl)
}

func (v Value) Index(i int) Value {
	switch k := v.kind(); k {
	case Array:
		tt := (*arrayType)(unsafe.Pointer(v.typ))
		if i < 0 || i > int(tt.len) {
			panic("reflect: array index out of range")
		}
		typ := tt.elem
		fl := v.flag & (flagRO | flagIndir | flagAddr)
		fl |= flag(typ.Kind()) << flagKindShift

		a := js.InternalObject(v.val)
		if fl&flagIndir != 0 && typ.Kind() != Array && typ.Kind() != Struct {
			return Value{typ, unsafe.Pointer(jsType(PtrTo(typ)).New(func() js.Object { return a.Index(i) }, func(v js.Object) { a.SetIndex(i, v) }).Unsafe()), fl}
		}
		return makeValue(typ, a.Index(i), fl)

	case Slice:
		s := js.InternalObject(v.iword())
		if i < 0 || i >= s.Length() {
			panic("reflect: slice index out of range")
		}
		tt := (*sliceType)(unsafe.Pointer(v.typ))
		typ := tt.elem
		fl := flagAddr | flagIndir | v.flag&flagRO
		fl |= flag(typ.Kind()) << flagKindShift

		i += s.Get("offset").Int()
		a := s.Get("array")
		if fl&flagIndir != 0 && typ.Kind() != Array && typ.Kind() != Struct {
			return Value{typ, unsafe.Pointer(jsType(PtrTo(typ)).New(func() js.Object { return a.Index(i) }, func(v js.Object) { a.SetIndex(i, v) }).Unsafe()), fl}
		}
		return makeValue(typ, a.Index(i), fl)

	case String:
		str := *(*string)(v.val)
		if i < 0 || i >= len(str) {
			panic("reflect: string index out of range")
		}
		fl := v.flag&flagRO | flag(Uint8<<flagKindShift)
		return Value{uint8Type, unsafe.Pointer(uintptr(str[i])), fl}

	default:
		panic(&ValueError{"reflect.Value.Index", k})
	}
}

func (v Value) IsNil() bool {
	switch k := v.kind(); k {
	case Chan, Ptr, Slice:
		return v.iword() == iword(jsType(v.typ).Get("nil").Unsafe())
	case Func:
		return v.iword() == iword(js.Global.Get("go$throwNilPointerError").Unsafe())
	case Map:
		return v.iword() == iword(js.InternalObject(false).Unsafe())
	case Interface:
		return js.InternalObject(v.iword()).IsNull()
	default:
		panic(&ValueError{"reflect.Value.IsNil", k})
	}
}

func (v Value) Len() int {
	switch k := v.kind(); k {
	case Array, Slice, String:
		return js.InternalObject(v.iword()).Length()
	// case Chan:
	// 	return chanlen(v.iword())
	case Map:
		return js.Global.Call("go$keys", v.iword()).Length()
	default:
		panic(&ValueError{"reflect.Value.Len", k})
	}
}

func (v Value) Pointer() uintptr {
	switch k := v.kind(); k {
	case Chan, Map, Ptr, Slice, UnsafePointer:
		if v.IsNil() {
			return 0
		}
		return uintptr(unsafe.Pointer(v.iword()))
	case Func:
		if v.IsNil() {
			return 0
		}
		return 1
	default:
		panic(&ValueError{"reflect.Value.Pointer", k})
	}
}

func (v Value) Set(x Value) {
	v.mustBeAssignable()
	x.mustBeExported()
	if v.flag&flagIndir != 0 {
		switch v.typ.Kind() {
		case Array:
			js.Global.Call("go$copyArray", v.val, x.val)
		case Interface:
			js.InternalObject(v.val).Call("go$set", js.InternalObject(valueInterface(x, false)))
		case Struct:
			copyStruct(js.InternalObject(v.val), js.InternalObject(x.val), v.typ)
		default:
			js.InternalObject(v.val).Call("go$set", x.iword())
		}
		return
	}
	v.val = x.val
}

func (v Value) SetCap(n int) {
	v.mustBeAssignable()
	v.mustBe(Slice)
	s := js.InternalObject(v.val).Call("go$get")
	if n < s.Length() || n > s.Get("capacity").Int() {
		panic("reflect: slice capacity out of range in SetCap")
	}
	newSlice := jsType(v.typ).New(s.Get("array"))
	newSlice.Set("offset", s.Get("offset"))
	newSlice.Set("length", s.Get("length"))
	newSlice.Set("capacity", n)
	js.InternalObject(v.val).Call("go$set", newSlice)
}

func (v Value) SetLen(n int) {
	v.mustBeAssignable()
	v.mustBe(Slice)
	s := js.InternalObject(v.val).Call("go$get")
	if n < 0 || n > s.Get("capacity").Int() {
		panic("reflect: slice length out of range in SetLen")
	}
	newSlice := jsType(v.typ).New(s.Get("array"))
	newSlice.Set("offset", s.Get("offset"))
	newSlice.Set("length", n)
	newSlice.Set("capacity", s.Get("capacity"))
	js.InternalObject(v.val).Call("go$set", newSlice)
}

func (v Value) Slice(i, j int) Value {
	var (
		cap int
		typ Type
		s   js.Object
	)
	switch kind := v.kind(); kind {
	case Array:
		if v.flag&flagAddr == 0 {
			panic("reflect.Value.Slice: slice of unaddressable array")
		}
		tt := (*arrayType)(unsafe.Pointer(v.typ))
		cap = int(tt.len)
		typ = SliceOf(tt.elem)
		s = jsType(typ).New(v.iword())

	case Slice:
		typ = v.typ
		s = js.InternalObject(v.iword())
		cap = s.Get("capacity").Int()

	case String:
		str := *(*string)(v.val)
		if i < 0 || j < i || j > len(str) {
			panic("reflect.Value.Slice: string slice index out of bounds")
		}
		return ValueOf(str[i:j])

	default:
		panic(&ValueError{"reflect.Value.Slice", kind})
	}

	if i < 0 || j < i || j > cap {
		panic("reflect.Value.Slice: slice index out of bounds")
	}

	return makeValue(typ, js.Global.Call("go$subslice", s, i, j), v.flag&flagRO)
}

func (v Value) Slice3(i, j, k int) Value {
	var (
		cap int
		typ Type
		s   js.Object
	)
	switch kind := v.kind(); kind {
	case Array:
		if v.flag&flagAddr == 0 {
			panic("reflect.Value.Slice: slice of unaddressable array")
		}
		tt := (*arrayType)(unsafe.Pointer(v.typ))
		cap = int(tt.len)
		typ = SliceOf(tt.elem)
		s = jsType(typ).New(v.iword())

	case Slice:
		typ = v.typ
		s = js.InternalObject(v.iword())
		cap = s.Get("capacity").Int()

	default:
		panic(&ValueError{"reflect.Value.Slice3", kind})
	}

	if i < 0 || j < i || k < j || k > cap {
		panic("reflect.Value.Slice3: slice index out of bounds")
	}

	return makeValue(typ, js.Global.Call("go$subslice", s, i, j, k), v.flag&flagRO)
}
