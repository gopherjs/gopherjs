// +build js

package reflect

import (
	"runtime"
	"unsafe"

	"github.com/gopherjs/gopherjs/js"
)

var initialized = false

func init() {
	// avoid dead code elimination
	used := func(i interface{}) {}
	used(rtype{})
	used(uncommonType{})
	used(method{})
	used(arrayType{})
	used(chanType{})
	used(funcType{})
	used(interfaceType{})
	used(mapType{})
	used(ptrType{})
	used(sliceType{})
	used(structType{})
	used(imethod{})
	used(structField{})

	pkg := js.Global.Get("$pkg")
	pkg.Set("kinds", map[string]Kind{
		"Bool":          Bool,
		"Int":           Int,
		"Int8":          Int8,
		"Int16":         Int16,
		"Int32":         Int32,
		"Int64":         Int64,
		"Uint":          Uint,
		"Uint8":         Uint8,
		"Uint16":        Uint16,
		"Uint32":        Uint32,
		"Uint64":        Uint64,
		"Uintptr":       Uintptr,
		"Float32":       Float32,
		"Float64":       Float64,
		"Complex64":     Complex64,
		"Complex128":    Complex128,
		"Array":         Array,
		"Chan":          Chan,
		"Func":          Func,
		"Interface":     Interface,
		"Map":           Map,
		"Ptr":           Ptr,
		"Slice":         Slice,
		"String":        String,
		"Struct":        Struct,
		"UnsafePointer": UnsafePointer,
	})
	pkg.Set("RecvDir", RecvDir)
	pkg.Set("SendDir", SendDir)
	pkg.Set("BothDir", BothDir)
	js.Global.Set("$reflect", pkg)
	initialized = true
	uint8Type = TypeOf(uint8(0)).(*rtype) // set for real
}

func jsType(typ Type) js.Object {
	return js.InternalObject(typ).Get("jsType")
}

func reflectType(typ js.Object) *rtype {
	return (*rtype)(unsafe.Pointer(typ.Call("reflectType").Unsafe()))
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

func makeValue(t Type, v js.Object, fl flag) Value {
	rt := t.common()
	if t.Kind() == Array || t.Kind() == Struct || t.Kind() == Ptr {
		return Value{rt, unsafe.Pointer(v.Unsafe()), fl | flag(t.Kind())}
	}
	return Value{rt, unsafe.Pointer(js.Global.Call("$newDataPointer", v, jsType(rt.ptrTo())).Unsafe()), fl | flag(t.Kind()) | flagIndir}
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

	return makeValue(typ, jsType(typ).Call("make", len, cap, func() js.Object { return jsType(typ.Elem()).Call("zero") }), 0)
}

func jsObject() *rtype {
	return reflectType(js.Global.Get("$packages").Get("github.com/gopherjs/gopherjs/js").Get("Object"))
}

func TypeOf(i interface{}) Type {
	if !initialized { // avoid error of uint8Type
		return &rtype{}
	}
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
		return Value{jsObject(), unsafe.Pointer(js.InternalObject(i).Unsafe()), flag(Interface)}
	}
	return makeValue(reflectType(c), js.InternalObject(i).Get("$val"), 0)
}

func arrayOf(count int, elem Type) Type {
	return reflectType(js.Global.Call("$arrayType", jsType(elem), count))
}

func ChanOf(dir ChanDir, t Type) Type {
	return reflectType(js.Global.Call("$chanType", jsType(t), dir == SendDir, dir == RecvDir))
}

func MapOf(key, elem Type) Type {
	switch key.Kind() {
	case Func, Map, Slice:
		panic("reflect.MapOf: invalid key type " + key.String())
	}

	return reflectType(js.Global.Call("$mapType", jsType(key), jsType(elem)))
}

func (t *rtype) ptrTo() *rtype {
	return reflectType(js.Global.Call("$ptrType", jsType(t)))
}

func SliceOf(t Type) Type {
	return reflectType(js.Global.Call("$sliceType", jsType(t)))
}

func Zero(typ Type) Value {
	return makeValue(typ, jsType(typ).Call("zero"), 0)
}

func unsafe_New(typ *rtype) unsafe.Pointer {
	switch typ.Kind() {
	case Struct:
		return unsafe.Pointer(jsType(typ).Get("Ptr").New().Unsafe())
	case Array:
		return unsafe.Pointer(jsType(typ).Call("zero").Unsafe())
	default:
		return unsafe.Pointer(js.Global.Call("$newDataPointer", jsType(typ).Call("zero"), jsType(typ.ptrTo())).Unsafe())
	}
}

func makeInt(f flag, bits uint64, t Type) Value {
	typ := t.common()
	ptr := unsafe_New(typ)
	switch typ.Kind() {
	case Int8:
		*(*int8)(ptr) = int8(bits)
	case Int16:
		*(*int16)(ptr) = int16(bits)
	case Int, Int32:
		*(*int32)(ptr) = int32(bits)
	case Int64:
		*(*int64)(ptr) = int64(bits)
	case Uint8:
		*(*uint8)(ptr) = uint8(bits)
	case Uint16:
		*(*uint16)(ptr) = uint16(bits)
	case Uint, Uint32, Uintptr:
		*(*uint32)(ptr) = uint32(bits)
	case Uint64:
		*(*uint64)(ptr) = uint64(bits)
	}
	return Value{typ, ptr, f | flagIndir | flag(typ.Kind())}
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
			args[i] = makeValue(argType, js.Arguments[i], 0)
		}
		resultsSlice := fn(args)
		switch ftyp.NumOut() {
		case 0:
			return nil
		case 1:
			return resultsSlice[0].object()
		default:
			results := js.Global.Get("Array").New(ftyp.NumOut())
			for i, r := range resultsSlice {
				results.SetIndex(i, r.object())
			}
			return results
		}
	}

	return Value{t, unsafe.Pointer(js.InternalObject(fv).Unsafe()), flag(Func)}
}

func memmove(adst, asrc unsafe.Pointer, n uintptr) {
	js.InternalObject(adst).Call("$set", js.InternalObject(asrc).Call("$get"))
}

func loadScalar(p unsafe.Pointer, n uintptr) uintptr {
	return js.InternalObject(p).Call("$get").Unsafe()
}

func makechan(typ *rtype, size uint64) (ch unsafe.Pointer) {
	return unsafe.Pointer(jsType(typ).New().Unsafe())
}

func makemap(t *rtype) (m unsafe.Pointer) {
	return unsafe.Pointer(js.Global.Get("$Map").New().Unsafe())
}

func mapaccess(t *rtype, m, key unsafe.Pointer) unsafe.Pointer {
	k := js.InternalObject(key).Call("$get")
	if !k.Get("$key").IsUndefined() {
		k = k.Call("$key")
	}
	entry := js.InternalObject(m).Get(k.Str())
	if entry.IsUndefined() {
		return nil
	}
	return unsafe.Pointer(js.Global.Call("$newDataPointer", entry.Get("v"), jsType(PtrTo(t.Elem()))).Unsafe())
}

func mapassign(t *rtype, m, key, val unsafe.Pointer) {
	kv := js.InternalObject(key).Call("$get")
	k := kv
	if !k.Get("$key").IsUndefined() {
		k = k.Call("$key")
	}
	jsVal := js.InternalObject(val).Call("$get")
	et := t.Elem()
	if et.Kind() == Struct {
		newVal := jsType(et).Call("zero")
		copyStruct(newVal, jsVal, et)
		jsVal = newVal
	}
	entry := js.Global.Get("Object").New()
	entry.Set("k", kv)
	entry.Set("v", jsVal)
	js.InternalObject(m).Set(k.Str(), entry)
}

func mapdelete(t *rtype, m unsafe.Pointer, key unsafe.Pointer) {
	k := js.InternalObject(key).Call("$get")
	if !k.Get("$key").IsUndefined() {
		k = k.Call("$key")
	}
	js.InternalObject(m).Delete(k.Str())
}

type mapIter struct {
	t    Type
	m    js.Object
	keys js.Object
	i    int
}

func mapiterinit(t *rtype, m unsafe.Pointer) *byte {
	return (*byte)(unsafe.Pointer(&mapIter{t, js.InternalObject(m), js.Global.Call("$keys", js.InternalObject(m)), 0}))
}

func mapiterkey(it *byte) unsafe.Pointer {
	iter := (*mapIter)(unsafe.Pointer(it))
	k := iter.keys.Index(iter.i)
	return unsafe.Pointer(js.Global.Call("$newDataPointer", iter.m.Get(k.Str()).Get("k"), jsType(PtrTo(iter.t.Key()))).Unsafe())
}

func mapiternext(it *byte) {
	iter := (*mapIter)(unsafe.Pointer(it))
	iter.i++
}

func maplen(m unsafe.Pointer) int {
	return js.Global.Call("$keys", js.InternalObject(m)).Length()
}

func cvtDirect(v Value, typ Type) Value {
	var srcVal = v.object()
	if srcVal == jsType(v.typ).Get("nil") {
		return makeValue(typ, jsType(typ).Get("nil"), v.flag)
	}

	var val js.Object
	switch k := typ.Kind(); k {
	case Chan:
		val = jsType(typ).New()
	case Slice:
		slice := jsType(typ).New(srcVal.Get("$array"))
		slice.Set("$offset", srcVal.Get("$offset"))
		slice.Set("$length", srcVal.Get("$length"))
		slice.Set("$capacity", srcVal.Get("$capacity"))
		val = js.Global.Call("$newDataPointer", slice, jsType(PtrTo(typ)))
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
		val = jsType(typ).New(srcVal.Get("$get"), srcVal.Get("$set"))
	case Struct:
		val = jsType(typ).Get("Ptr").New()
		copyStruct(val, srcVal, typ)
	case Array, Func, Interface, Map, String:
		val = js.InternalObject(v.ptr)
	default:
		panic(&ValueError{"reflect.Convert", k})
	}
	return Value{typ.common(), unsafe.Pointer(val.Unsafe()), v.flag&(flagRO|flagIndir) | flag(typ.Kind())}
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

	dstVal := dst.object()
	if dk == Array {
		dstVal = jsType(SliceOf(dst.typ.Elem())).New(dstVal)
	}

	srcVal := src.object()
	if sk == Array {
		srcVal = jsType(SliceOf(src.typ.Elem())).New(srcVal)
	}

	return js.Global.Call("$copySlice", dstVal, srcVal).Int()
}

func methodReceiver(op string, v Value, i int) (rcvrtype, t *rtype, fn unsafe.Pointer) { // TODO cleanup
	var name string
	if v.typ.Kind() == Interface {
		tt := (*interfaceType)(unsafe.Pointer(v.typ))
		if i < 0 || i >= len(tt.methods) {
			panic("reflect: internal error: invalid method index")
		}
		m := &tt.methods[i]
		if m.pkgPath != nil {
			panic("reflect: " + op + " of unexported method")
		}
		iface := (*nonEmptyInterface)(v.ptr)
		if iface.itab == nil {
			panic("reflect: " + op + " of method on nil interface value")
		}
		// rcvrtype = iface.itab.typ
		t = m.typ
		name = *m.name
	} else {
		// rcvrtype = v.typ
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
	rcvr := v.object()
	if isWrapped(v.typ) {
		rcvr = jsType(v.typ).New(rcvr)
	}
	fn = unsafe.Pointer(rcvr.Get(name).Unsafe())
	return
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
		return interface{}(unsafe.Pointer(jsType(v.typ).New(v.object()).Unsafe()))
	}
	return interface{}(unsafe.Pointer(v.object().Unsafe()))
}

func ifaceE2I(t *rtype, src interface{}, dst unsafe.Pointer) {
	js.InternalObject(dst).Call("$set", js.InternalObject(src))
}

func methodName() string {
	return "?FIXME?"
}

func makeMethodValue(op string, v Value) Value {
	if v.flag&flagMethod == 0 {
		panic("reflect: internal error: invalid use of makePartialFunc")
	}

	_, _, fn := methodReceiver(op, v, int(v.flag)>>flagMethodShift)
	rcvr := v.object()
	if isWrapped(v.typ) {
		rcvr = jsType(v.typ).New(rcvr)
	}
	fv := func() js.Object {
		return js.InternalObject(fn).Call("apply", rcvr, js.Arguments)
	}
	return Value{v.Type().common(), unsafe.Pointer(js.InternalObject(fv).Unsafe()), v.flag&flagRO | flag(Func)}
}

func (t *rtype) pointers() bool {
	switch t.Kind() {
	case Ptr, Map, Chan, Func, Struct, Array:
		return true
	default:
		return false
	}
}

func (t *rtype) Comparable() bool {
	switch t.Kind() {
	case Func, Slice, Map:
		return false
	case Array:
		return t.Elem().Comparable()
	case Struct:
		for i := 0; i < t.NumField(); i++ {
			if !t.Field(i).Type.Comparable() {
				return false
			}
		}
	}
	return true
}

func (t *uncommonType) Method(i int) (m Method) {
	if t == nil || i < 0 || i >= len(t.methods) {
		panic("reflect: Method index out of range")
	}
	p := &t.methods[i]
	if p.name != nil {
		m.Name = *p.name
	}
	fl := flag(Func)
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

func (v Value) object() js.Object {
	if v.typ.Kind() == Array || v.typ.Kind() == Struct {
		return js.InternalObject(v.ptr)
	}
	if v.flag&flagIndir != 0 {
		val := js.InternalObject(v.ptr).Call("$get")
		if val != js.Global.Get("$ifaceNil") && val.Get("constructor") != jsType(v.typ) {
			switch v.typ.Kind() {
			case Uint64, Int64:
				val = jsType(v.typ).New(val.Get("$high"), val.Get("$low"))
			case Complex64, Complex128:
				val = jsType(v.typ).New(val.Get("$real"), val.Get("$imag"))
			case Slice:
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

func (v Value) call(op string, in []Value) []Value {
	t := v.typ
	var (
		fn   unsafe.Pointer
		rcvr js.Object
	)
	if v.flag&flagMethod != 0 {
		_, t, fn = methodReceiver(op, v, int(v.flag)>>flagMethodShift)
		rcvr = v.object()
		if isWrapped(v.typ) {
			rcvr = jsType(v.typ).New(rcvr)
		}
	} else {
		fn = unsafe.Pointer(v.object().Unsafe())
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
		argsArray.SetIndex(i, arg.assignTo("reflect.Value.Call", t.In(i).common(), nil).object())
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
	case Chan, Slice:
		return v.object().Get("$capacity").Int()
	}
	panic(&ValueError{"reflect.Value.Cap", k})
}

func (v Value) Elem() Value {
	switch k := v.kind(); k {
	case Interface:
		val := v.object()
		if val == js.Global.Get("$ifaceNil") {
			return Value{}
		}
		typ := reflectType(val.Get("constructor"))
		return makeValue(typ, val.Get("$val"), v.flag&flagRO)

	case Ptr:
		if v.IsNil() {
			return Value{}
		}
		val := v.object()
		tt := (*ptrType)(unsafe.Pointer(v.typ))
		fl := v.flag&flagRO | flagIndir | flagAddr
		fl |= flag(tt.elem.Kind())
		return Value{tt.elem, unsafe.Pointer(val.Unsafe()), fl}

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
	fl |= flag(typ.Kind())

	s := js.InternalObject(v.ptr)
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
		fl |= flag(typ.Kind())

		a := js.InternalObject(v.ptr)
		if fl&flagIndir != 0 && typ.Kind() != Array && typ.Kind() != Struct {
			return Value{typ, unsafe.Pointer(jsType(PtrTo(typ)).New(func() js.Object { return a.Index(i) }, func(v js.Object) { a.SetIndex(i, v) }).Unsafe()), fl}
		}
		return makeValue(typ, a.Index(i), fl)

	case Slice:
		s := v.object()
		if i < 0 || i >= s.Get("$length").Int() {
			panic("reflect: slice index out of range")
		}
		tt := (*sliceType)(unsafe.Pointer(v.typ))
		typ := tt.elem
		fl := flagAddr | flagIndir | v.flag&flagRO
		fl |= flag(typ.Kind())

		i += s.Get("$offset").Int()
		a := s.Get("$array")
		if fl&flagIndir != 0 && typ.Kind() != Array && typ.Kind() != Struct {
			return Value{typ, unsafe.Pointer(jsType(PtrTo(typ)).New(func() js.Object { return a.Index(i) }, func(v js.Object) { a.SetIndex(i, v) }).Unsafe()), fl}
		}
		return makeValue(typ, a.Index(i), fl)

	case String:
		str := *(*string)(v.ptr)
		if i < 0 || i >= len(str) {
			panic("reflect: string index out of range")
		}
		fl := v.flag&flagRO | flag(Uint8)
		c := str[i]
		return Value{uint8Type, unsafe.Pointer(&c), fl | flagIndir}

	default:
		panic(&ValueError{"reflect.Value.Index", k})
	}
}

func (v Value) IsNil() bool {
	switch k := v.kind(); k {
	case Chan, Ptr, Slice:
		return v.object() == jsType(v.typ).Get("nil")
	case Func:
		return v.object() == js.Global.Get("$throwNilPointerError")
	case Map:
		return v.object() == js.InternalObject(false)
	case Interface:
		return v.object() == js.Global.Get("$ifaceNil")
	default:
		panic(&ValueError{"reflect.Value.IsNil", k})
	}
}

func (v Value) Len() int {
	switch k := v.kind(); k {
	case Array, String:
		return v.object().Length()
	case Slice:
		return v.object().Get("$length").Int()
	case Chan:
		return v.object().Get("$buffer").Get("length").Int()
	case Map:
		return js.Global.Call("$keys", v.object()).Length()
	default:
		panic(&ValueError{"reflect.Value.Len", k})
	}
}

func (v Value) Pointer() uintptr {
	switch k := v.kind(); k {
	case Chan, Map, Ptr, UnsafePointer:
		if v.IsNil() {
			return 0
		}
		return v.object().Unsafe()
	case Func:
		if v.IsNil() {
			return 0
		}
		return 1
	case Slice:
		if v.IsNil() {
			return 0
		}
		return v.object().Get("$array").Unsafe()
	default:
		panic(&ValueError{"reflect.Value.Pointer", k})
	}
}

func (v Value) Set(x Value) {
	v.mustBeAssignable()
	x.mustBeExported()
	x = x.assignTo("reflect.Set", v.typ, nil)
	if v.flag&flagIndir != 0 {
		switch v.typ.Kind() {
		case Array:
			js.Global.Call("$copy", js.InternalObject(v.ptr), js.InternalObject(x.ptr), jsType(v.typ))
		case Interface:
			js.InternalObject(v.ptr).Call("$set", js.InternalObject(valueInterface(x, false)))
		case Struct:
			copyStruct(js.InternalObject(v.ptr), js.InternalObject(x.ptr), v.typ)
		default:
			js.InternalObject(v.ptr).Call("$set", x.object())
		}
		return
	}
	v.ptr = x.ptr
}

func (v Value) SetCap(n int) {
	v.mustBeAssignable()
	v.mustBe(Slice)
	s := js.InternalObject(v.ptr).Call("$get")
	if n < s.Get("$length").Int() || n > s.Get("$capacity").Int() {
		panic("reflect: slice capacity out of range in SetCap")
	}
	newSlice := jsType(v.typ).New(s.Get("$array"))
	newSlice.Set("$offset", s.Get("$offset"))
	newSlice.Set("$length", s.Get("$length"))
	newSlice.Set("$capacity", n)
	js.InternalObject(v.ptr).Call("$set", newSlice)
}

func (v Value) SetLen(n int) {
	v.mustBeAssignable()
	v.mustBe(Slice)
	s := js.InternalObject(v.ptr).Call("$get")
	if n < 0 || n > s.Get("$capacity").Int() {
		panic("reflect: slice length out of range in SetLen")
	}
	newSlice := jsType(v.typ).New(s.Get("$array"))
	newSlice.Set("$offset", s.Get("$offset"))
	newSlice.Set("$length", n)
	newSlice.Set("$capacity", s.Get("$capacity"))
	js.InternalObject(v.ptr).Call("$set", newSlice)
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
		s = jsType(typ).New(v.object())

	case Slice:
		typ = v.typ
		s = v.object()
		cap = s.Get("$capacity").Int()

	case String:
		str := *(*string)(v.ptr)
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

	return makeValue(typ, js.Global.Call("$subslice", s, i, j), v.flag&flagRO)
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
		s = jsType(typ).New(v.object())

	case Slice:
		typ = v.typ
		s = v.object()
		cap = s.Get("$capacity").Int()

	default:
		panic(&ValueError{"reflect.Value.Slice3", kind})
	}

	if i < 0 || j < i || k < j || k > cap {
		panic("reflect.Value.Slice3: slice index out of bounds")
	}

	return makeValue(typ, js.Global.Call("$subslice", s, i, j, k), v.flag&flagRO)
}

func (v Value) Close() {
	v.mustBe(Chan)
	v.mustBeExported()
	js.Global.Call("$close", v.object())
}

func (v Value) TrySend(x Value) bool {
	v.mustBe(Chan)
	v.mustBeExported()
	tt := (*chanType)(unsafe.Pointer(v.typ))
	if ChanDir(tt.dir)&SendDir == 0 {
		panic("reflect: send on recv-only channel")
	}
	x.mustBeExported()

	c := v.object()
	if !c.Get("$closed").Bool() && c.Get("$recvQueue").Length() == 0 && c.Get("$buffer").Length() == c.Get("$capacity").Int() {
		return false
	}
	x = x.assignTo("reflect.Value.Send", tt.elem, nil)
	js.Global.Call("$send", c, x.object())
	return true
}

func (v Value) Send(x Value) {
	panic(&runtime.NotSupportedError{"reflect.Value.Send, use reflect.Value.TrySend is possible"})
}

func (v Value) TryRecv() (x Value, ok bool) {
	v.mustBe(Chan)
	v.mustBeExported()
	tt := (*chanType)(unsafe.Pointer(v.typ))
	if ChanDir(tt.dir)&RecvDir == 0 {
		panic("reflect: recv on send-only channel")
	}

	res := js.Global.Call("$recv", v.object())
	if res.Get("constructor") == js.Global.Get("Function") {
		return Value{}, false
	}
	return makeValue(tt.elem, res.Index(0), 0), res.Index(1).Bool()
}

func (v Value) Recv() (x Value, ok bool) {
	panic(&runtime.NotSupportedError{"reflect.Value.Recv, use reflect.Value.TryRecv is possible"})
}

func DeepEqual(a1, a2 interface{}) bool {
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

func deepValueEqualJs(v1, v2 Value, visited [][2]unsafe.Pointer) bool {
	if !v1.IsValid() || !v2.IsValid() {
		return !v1.IsValid() && !v2.IsValid()
	}
	if v1.Type() != v2.Type() {
		return false
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
		var n = v1.Len()
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
		var n = v1.NumField()
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
		var keys = v1.MapKeys()
		if len(keys) != v2.Len() {
			return false
		}
		for _, k := range keys {
			if !deepValueEqualJs(v1.MapIndex(k), v2.MapIndex(k), visited) {
				return false
			}
		}
		return true
	case Func:
		return v1.IsNil() && v2.IsNil()
	}

	return js.Global.Call("$interfaceIsEqual", js.InternalObject(valueInterface(v1, false)), js.InternalObject(valueInterface(v2, false))).Bool()
}
