//go:build js

package reflectlite

import (
	"unsafe"

	"internal/abi"

	"github.com/gopherjs/gopherjs/js"
)

var initialized = false

func init() {
	// avoid dead code elimination
	used := func(i any) {}
	used(rtype{})
	used(arrayType{})
	used(chanType{})
	used(funcType{})
	used(interfaceType{})
	used(mapType{})
	used(ptrType{})
	used(sliceType{})
	used(structType{})
	used(imethod{})

	initialized = true
	uint8Type = TypeOf(uint8(0)).(*rtype) // set for real
}

var uint8Type *rtype

var (
	idJsType      = "_jsType"
	idReflectType = "_reflectType"
	idKindType    = "kindType"
	idRtype       = "_rtype"
)

func jsType(typ *abi.Type) *js.Object {
	return js.InternalObject(typ).Get(idJsType)
}

func jsPtrTo(typ *abi.Type) *js.Object {
	return jsType(PtrTo(toRType(typ)).(*rtype).Type)
}

func reflectType(typ *js.Object) *rtype {
	if typ.Get(idReflectType) == js.Undefined {
		rt := &rtype{
			Size: uintptr(typ.Get("size").Int()),
			Kind: uint8(typ.Get("kind").Int()),
			Str:  newNameOff(newName(internalStr(typ.Get("string")), "", typ.Get("exported").Bool(), false)),
		}
		js.InternalObject(rt).Set(idJsType, typ)
		typ.Set(idReflectType, js.InternalObject(rt))

		methodSet := js.Global.Call("$methodSet", typ)
		if methodSet.Length() != 0 || typ.Get("named").Bool() {
			rt.TFlag |= abi.TFlagUncommon
			if typ.Get("named").Bool() {
				rt.TFlag |= abi.TFlagNamed
			}
			var reflectMethods []abi.Method
			for i := 0; i < methodSet.Length(); i++ { // Exported methods first.
				m := methodSet.Index(i)
				exported := internalStr(m.Get("pkg")) == ""
				if !exported {
					continue
				}
				reflectMethods = append(reflectMethods, abi.Method{
					Name: newNameOff(newName(internalStr(m.Get("name")), "", exported, false)),
					Mtyp: newTypeOff(reflectType(m.Get("typ"))),
				})
			}
			xcount := uint16(len(reflectMethods))
			for i := 0; i < methodSet.Length(); i++ { // Unexported methods second.
				m := methodSet.Index(i)
				exported := internalStr(m.Get("pkg")) == ""
				if exported {
					continue
				}
				reflectMethods = append(reflectMethods, abi.Method{
					Name: newNameOff(newName(internalStr(m.Get("name")), "", exported, false)),
					Mtyp: newTypeOff(reflectType(m.Get("typ"))),
				})
			}
			ut := &abi.UncommonType{
				PkgPath:  newNameOff(newName(internalStr(typ.Get("pkg")), "", false, false)),
				Mcount:   uint16(methodSet.Length()),
				Xcount:   xcount,
				Methods_: reflectMethods,
			}
			abi.UncommonTypeMap[rt] = ut
			js.InternalObject(ut).Set(idJsType, typ)
		}

		switch rt.Kind() {
		case abi.Array:
			setKindType(rt, &arrayType{
				elem: reflectType(typ.Get("elem")),
				len:  uintptr(typ.Get("len").Int()),
			})
		case abi.Chan:
			dir := BothDir
			if typ.Get("sendOnly").Bool() {
				dir = SendDir
			}
			if typ.Get("recvOnly").Bool() {
				dir = RecvDir
			}
			setKindType(rt, &chanType{
				elem: reflectType(typ.Get("elem")),
				dir:  uintptr(dir),
			})
		case abi.Func:
			params := typ.Get("params")
			in := make([]*rtype, params.Length())
			for i := range in {
				in[i] = reflectType(params.Index(i))
			}
			results := typ.Get("results")
			out := make([]*rtype, results.Length())
			for i := range out {
				out[i] = reflectType(results.Index(i))
			}
			outCount := uint16(results.Length())
			if typ.Get("variadic").Bool() {
				outCount |= 1 << 15
			}
			setKindType(rt, &abi.FuncType{
				Type:     *rt,
				InCount:  uint16(params.Length()),
				OutCount: outCount,
				In_:      in,
				Out_:     out,
			})
		case abi.Interface:
			methods := typ.Get("methods")
			imethods := make([]imethod, methods.Length())
			for i := range imethods {
				m := methods.Index(i)
				imethods[i] = imethod{
					name: newNameOff(newName(internalStr(m.Get("name")), "", internalStr(m.Get("pkg")) == "", false)),
					typ:  newTypeOff(reflectType(m.Get("typ"))),
				}
			}
			setKindType(rt, &interfaceType{
				rtype:   *rt,
				pkgPath: newName(internalStr(typ.Get("pkg")), "", false, false),
				methods: imethods,
			})
		case abi.Map:
			setKindType(rt, &mapType{
				key:  reflectType(typ.Get("key")),
				elem: reflectType(typ.Get("elem")),
			})
		case abi.Pointer:
			setKindType(rt, &ptrType{
				elem: reflectType(typ.Get("elem")),
			})
		case abi.Slice:
			setKindType(rt, &sliceType{
				elem: reflectType(typ.Get("elem")),
			})
		case abi.Struct:
			fields := typ.Get("fields")
			reflectFields := make([]abi.StructField, fields.Length())
			for i := range reflectFields {
				f := fields.Index(i)
				reflectFields[i] = abi.StructField{
					Name:   newName(internalStr(f.Get("name")), internalStr(f.Get("tag")), f.Get("exported").Bool(), f.Get("embedded").Bool()),
					Typ:    reflectType(f.Get("typ")),
					Offset: uintptr(i),
				}
			}
			setKindType(rt, &structType{
				rtype:   *rt,
				pkgPath: newName(internalStr(typ.Get("pkgPath")), "", false, false),
				fields:  reflectFields,
			})
		}
	}

	return (*rtype)(unsafe.Pointer(typ.Get(idReflectType).Unsafe()))
}

func setKindType(rt *rtype, kindType any) {
	js.InternalObject(rt).Set(idKindType, js.InternalObject(kindType))
	js.InternalObject(kindType).Set(idRtype, js.InternalObject(rt))
}

type nameData struct {
	name     string
	tag      string
	exported bool
	embedded bool
}

var nameMap = make(map[*byte]*nameData)

func (n name) name() (s string) { return nameMap[n.bytes].name }
func (n name) tag() (s string)  { return nameMap[n.bytes].tag }
func (n name) isExported() bool { return nameMap[n.bytes].exported }
func (n name) embedded() bool   { return nameMap[n.bytes].embedded }

func newName(n, tag string, exported, embedded bool) name {
	b := new(byte)
	nameMap[b] = &nameData{
		name:     n,
		tag:      tag,
		exported: exported,
		embedded: embedded,
	}
	return name{
		bytes: b,
	}
}

func pkgPath(n abi.Name) string { return "" }

func internalStr(strObj *js.Object) string {
	var c struct{ str string }
	js.InternalObject(c).Set("str", strObj) // get string without internalizing
	return c.str
}

func isWrapped(typ *abi.Type) bool {
	return jsType(typ).Get("wrapped").Bool()
}

func copyStruct(dst, src *js.Object, typ *abi.Type) {
	fields := jsType(typ).Get("fields")
	for i := 0; i < fields.Length(); i++ {
		prop := fields.Index(i).Get("prop").String()
		dst.Set(prop, src.Get(prop))
	}
}

func makeValue(t *abi.Type, v *js.Object, fl flag) Value {
	rt := t.common()
	switch t.Kind() {
	case abi.Array, abi.Struct, abi.Pointer:
		return Value{rt, unsafe.Pointer(v.Unsafe()), fl | flag(t.Kind())}
	}
	return Value{rt, unsafe.Pointer(js.Global.Call("$newDataPointer", v, jsType(rt.ptrTo())).Unsafe()), fl | flag(t.Kind()) | flagIndir}
}

func MakeSlice(typ Type, len, cap int) Value {
	if typ.Kind() != abi.Slice {
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

	return makeValue(typ, js.Global.Call("$makeSlice", jsType(typ), len, cap, js.InternalObject(func() *js.Object { return jsType(typ.Elem()).Call("zero") })), 0)
}

func TypeOf(i any) Type {
	if !initialized { // avoid error of uint8Type
		return &rtype{}
	}
	if i == nil {
		return nil
	}
	return reflectType(js.InternalObject(i).Get("constructor"))
}

func ValueOf(i any) Value {
	if i == nil {
		return Value{}
	}
	return makeValue(reflectType(js.InternalObject(i).Get("constructor")), js.InternalObject(i).Get("$val"), 0)
}

func ArrayOf(count int, elem Type) Type {
	return reflectType(js.Global.Call("$arrayType", jsType(elem), count))
}

func ChanOf(dir ChanDir, t Type) Type {
	return reflectType(js.Global.Call("$chanType", jsType(t), dir == SendDir, dir == RecvDir))
}

func FuncOf(in, out []Type, variadic bool) Type {
	if variadic && (len(in) == 0 || in[len(in)-1].Kind() != abi.Slice) {
		panic("reflect.FuncOf: last arg of variadic func must be slice")
	}

	jsIn := make([]*js.Object, len(in))
	for i, v := range in {
		jsIn[i] = jsType(v)
	}
	jsOut := make([]*js.Object, len(out))
	for i, v := range out {
		jsOut[i] = jsType(v)
	}
	return reflectType(js.Global.Call("$funcType", jsIn, jsOut, variadic))
}

func MapOf(key, elem Type) Type {
	switch key.Kind() {
	case abi.Func, abi.Map, abi.Slice:
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

func unsafe_New(typ *abi.Type) unsafe.Pointer {
	switch typ.Kind() {
	case abi.Struct:
		return unsafe.Pointer(jsType(typ).Get("ptr").New().Unsafe())
	case abi.Array:
		return unsafe.Pointer(jsType(typ).Call("zero").Unsafe())
	default:
		return unsafe.Pointer(js.Global.Call("$newDataPointer", jsType(typ).Call("zero"), jsType(typ.ptrTo())).Unsafe())
	}
}

func makeInt(f flag, bits uint64, t Type) Value {
	typ := t.common()
	ptr := unsafe_New(typ)
	switch typ.Kind() {
	case abi.Int8:
		*(*int8)(ptr) = int8(bits)
	case abi.Int16:
		*(*int16)(ptr) = int16(bits)
	case abi.Int, abi.Int32:
		*(*int32)(ptr) = int32(bits)
	case abi.Int64:
		*(*int64)(ptr) = int64(bits)
	case abi.Uint8:
		*(*uint8)(ptr) = uint8(bits)
	case abi.Uint16:
		*(*uint16)(ptr) = uint16(bits)
	case abi.Uint, abi.Uint32, abi.Uintptr:
		*(*uint32)(ptr) = uint32(bits)
	case abi.Uint64:
		*(*uint64)(ptr) = uint64(bits)
	}
	return Value{typ, ptr, f | flagIndir | flag(typ.Kind())}
}

func MakeFunc(typ Type, fn func(args []Value) (results []Value)) Value {
	if typ.Kind() != abi.Func {
		panic("reflect: call of MakeFunc with non-Func type")
	}

	t := typ.common()
	ftyp := (*funcType)(unsafe.Pointer(t))

	fv := js.MakeFunc(func(this *js.Object, arguments []*js.Object) any {
		args := make([]Value, ftyp.NumIn())
		for i := range args {
			argType := ftyp.In(i).common()
			args[i] = makeValue(argType, arguments[i], 0)
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
	})

	return Value{t, unsafe.Pointer(fv.Unsafe()), flag(Func)}
}

func typedmemmove(t *abi.Type, dst, src unsafe.Pointer) {
	js.InternalObject(dst).Call("$set", js.InternalObject(src).Call("$get"))
}

func loadScalar(p unsafe.Pointer, n uintptr) uintptr {
	return js.InternalObject(p).Call("$get").Unsafe()
}

func makechan(typ *rtype, size int) (ch unsafe.Pointer) {
	ctyp := (*chanType)(unsafe.Pointer(typ))
	return unsafe.Pointer(js.Global.Get("$Chan").New(jsType(ctyp.elem), size).Unsafe())
}

func makemap(t *rtype, cap int) (m unsafe.Pointer) {
	return unsafe.Pointer(js.Global.Get("Map").New().Unsafe())
}

func keyFor(t *rtype, key unsafe.Pointer) (*js.Object, string) {
	kv := js.InternalObject(key)
	if kv.Get("$get") != js.Undefined {
		kv = kv.Call("$get")
	}
	k := jsType(t.Key()).Call("keyFor", kv).String()
	return kv, k
}

func mapaccess(t *rtype, m, key unsafe.Pointer) unsafe.Pointer {
	_, k := keyFor(t, key)
	entry := js.InternalObject(m).Call("get", k)
	if entry == js.Undefined {
		return nil
	}
	return unsafe.Pointer(js.Global.Call("$newDataPointer", entry.Get("v"), jsPtrTo(t.Elem())).Unsafe())
}

func mapassign(t *rtype, m, key, val unsafe.Pointer) {
	kv, k := keyFor(t, key)
	jsVal := js.InternalObject(val).Call("$get")
	et := t.Elem()
	if et.Kind() == abi.Struct {
		newVal := jsType(et).Call("zero")
		copyStruct(newVal, jsVal, et)
		jsVal = newVal
	}
	entry := js.Global.Get("Object").New()
	entry.Set("k", kv)
	entry.Set("v", jsVal)
	js.InternalObject(m).Call("set", k, entry)
}

func mapdelete(t *rtype, m unsafe.Pointer, key unsafe.Pointer) {
	_, k := keyFor(t, key)
	js.InternalObject(m).Call("delete", k)
}

type mapIter struct {
	t    Type
	m    *js.Object
	keys *js.Object
	i    int

	// last is the last object the iterator indicates. If this object exists, the functions that return the
	// current key or value returns this object, regardless of the current iterator. It is because the current
	// iterator might be stale due to key deletion in a loop.
	last *js.Object
}

func (iter *mapIter) skipUntilValidKey() {
	for iter.i < iter.keys.Length() {
		k := iter.keys.Index(iter.i)
		if iter.m.Call("get", k) != js.Undefined {
			break
		}
		// The key is already deleted. Move on the next item.
		iter.i++
	}
}

func mapiterinit(t *rtype, m unsafe.Pointer) unsafe.Pointer {
	return unsafe.Pointer(&mapIter{t, js.InternalObject(m), js.Global.Get("Array").Call("from", js.InternalObject(m).Call("keys")), 0, nil})
}

type TypeEx interface {
	Type
	Key() Type
}

func mapiterkey(it unsafe.Pointer) unsafe.Pointer {
	iter := (*mapIter)(it)
	var kv *js.Object
	if iter.last != nil {
		kv = iter.last
	} else {
		iter.skipUntilValidKey()
		if iter.i == iter.keys.Length() {
			return nil
		}
		k := iter.keys.Index(iter.i)
		kv = iter.m.Call("get", k)

		// Record the key-value pair for later accesses.
		iter.last = kv
	}
	return unsafe.Pointer(js.Global.Call("$newDataPointer", kv.Get("k"), jsPtrTo(iter.t.(TypeEx).Key())).Unsafe())
}

func mapiternext(it unsafe.Pointer) {
	iter := (*mapIter)(it)
	iter.last = nil
	iter.i++
}

func maplen(m unsafe.Pointer) int {
	return js.InternalObject(m).Get("size").Int()
}

func cvtDirect(v Value, typ Type) Value {
	srcVal := v.object()
	if srcVal == jsType(v.typ).Get("nil") {
		return makeValue(typ, jsType(typ).Get("nil"), v.flag)
	}

	var val *js.Object
	switch k := typ.Kind(); k {
	case abi.Slice:
		slice := jsType(typ).New(srcVal.Get("$array"))
		slice.Set("$offset", srcVal.Get("$offset"))
		slice.Set("$length", srcVal.Get("$length"))
		slice.Set("$capacity", srcVal.Get("$capacity"))
		val = js.Global.Call("$newDataPointer", slice, jsPtrTo(typ))
	case abi.Pointer:
		if typ.Elem().Kind() == abi.Struct {
			if typ.Elem() == v.typ.Elem() {
				val = srcVal
				break
			}
			val = jsType(typ).New()
			copyStruct(val, srcVal, typ.Elem())
			break
		}
		val = jsType(typ).New(srcVal.Get("$get"), srcVal.Get("$set"))
	case abi.Struct:
		val = jsType(typ).Get("ptr").New()
		copyStruct(val, srcVal, typ)
	case abi.Array, abi.Bool, abi.Chan, abi.Func, abi.Interface, abi.Map, abi.String:
		val = js.InternalObject(v.ptr)
	default:
		panic(&ValueError{"reflect.Convert", k})
	}
	return Value{typ.common(), unsafe.Pointer(val.Unsafe()), v.flag.ro() | v.flag&flagIndir | flag(typ.Kind())}
}

func Copy(dst, src Value) int {
	dk := dst.kind()
	if dk != abi.Array && dk != abi.Slice {
		panic(&ValueError{"reflect.Copy", dk})
	}
	if dk == abi.Array {
		dst.mustBeAssignable()
	}
	dst.mustBeExported()

	sk := src.kind()
	var stringCopy bool
	if sk != abi.Array && sk != abi.Slice {
		stringCopy = sk == abi.String && dst.typ.Elem().Kind() == abi.Uint8
		if !stringCopy {
			panic(&ValueError{"reflect.Copy", sk})
		}
	}
	src.mustBeExported()

	if !stringCopy {
		typesMustMatch("reflect.Copy", dst.typ.Elem(), src.typ.Elem())
	}

	dstVal := dst.object()
	if dk == abi.Array {
		dstVal = jsType(SliceOf(dst.typ.Elem())).New(dstVal)
	}

	srcVal := src.object()
	if sk == abi.Array {
		srcVal = jsType(SliceOf(src.typ.Elem())).New(srcVal)
	}

	if stringCopy {
		return js.Global.Call("$copyString", dstVal, srcVal).Int()
	}
	return js.Global.Call("$copySlice", dstVal, srcVal).Int()
}

func methodReceiver(op string, v Value, i int) (_ *rtype, t *funcType, fn unsafe.Pointer) {
	var prop string
	if v.typ.Kind() == abi.Interface {
		tt := (*interfaceType)(unsafe.Pointer(v.typ))
		if i < 0 || i >= len(tt.methods) {
			panic("reflect: internal error: invalid method index")
		}
		m := &tt.methods[i]
		if !tt.nameOff(m.name).isExported() {
			panic("reflect: " + op + " of unexported method")
		}
		t = (*funcType)(unsafe.Pointer(tt.typeOff(m.typ)))
		prop = tt.nameOff(m.name).name()
	} else {
		ms := v.typ.exportedMethods()
		if uint(i) >= uint(len(ms)) {
			panic("reflect: internal error: invalid method index")
		}
		m := ms[i]
		if !v.typ.nameOff(m.name).isExported() {
			panic("reflect: " + op + " of unexported method")
		}
		t = (*funcType)(unsafe.Pointer(v.typ.typeOff(m.mtyp)))
		prop = js.Global.Call("$methodSet", jsType(v.typ)).Index(i).Get("prop").String()
	}
	rcvr := v.object()
	if isWrapped(v.typ) {
		rcvr = jsType(v.typ).New(rcvr)
	}
	fn = unsafe.Pointer(rcvr.Get(prop).Unsafe())
	return
}

func valueInterface(v Value) any {
	if v.flag == 0 {
		panic(&ValueError{"reflect.Value.Interface", 0})
	}

	if v.flag&flagMethod != 0 {
		v = makeMethodValue("Interface", v)
	}

	if isWrapped(v.typ) {
		if v.flag&flagIndir != 0 && v.Kind() == abi.Struct {
			cv := jsType(v.typ).Call("zero")
			copyStruct(cv, v.object(), v.typ)
			return any(unsafe.Pointer(jsType(v.typ).New(cv).Unsafe()))
		}
		return any(unsafe.Pointer(jsType(v.typ).New(v.object()).Unsafe()))
	}
	return any(unsafe.Pointer(v.object().Unsafe()))
}

func ifaceE2I(t *abi.Type, src any, dst unsafe.Pointer) {
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
	fv := js.MakeFunc(func(this *js.Object, arguments []*js.Object) any {
		return js.InternalObject(fn).Call("apply", rcvr, arguments)
	})
	return Value{v.Type().common(), unsafe.Pointer(fv.Unsafe()), v.flag.ro() | flag(Func)}
}

var jsObjectPtr = reflectType(js.Global.Get("$jsObjectPtr"))

func wrapJsObject(typ Type, val *js.Object) *js.Object {
	if typ == jsObjectPtr {
		return jsType(jsObjectPtr).New(val)
	}
	return val
}

func unwrapJsObject(typ Type, val *js.Object) *js.Object {
	if typ == jsObjectPtr {
		return val.Get("object")
	}
	return val
}

func getJsTag(tag string) string {
	for tag != "" {
		// skip leading space
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// scan to colon.
		// a space or a quote is a syntax error
		i = 0
		for i < len(tag) && tag[i] != ' ' && tag[i] != ':' && tag[i] != '"' {
			i++
		}
		if i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		// scan quoted string to find value
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		if name == "js" {
			value, _ := unquote(qvalue)
			return value
		}
	}
	return ""
}

// PtrTo returns the pointer type with element t.
// For example, if t represents type Foo, PtrTo(t) represents *Foo.
func PtrTo(t Type) Type {
	return t.(*rtype).ptrTo()
}

// copyVal returns a Value containing the map key or value at ptr,
// allocating a new variable as needed.
func copyVal(typ *rtype, fl flag, ptr unsafe.Pointer) Value {
	if ifaceIndir(typ) {
		// Copy result so future changes to the map
		// won't change the underlying value.
		c := unsafe_New(typ)
		typedmemmove(typ, c, ptr)
		return Value{typ, c, fl | flagIndir}
	}
	return Value{typ, *(*unsafe.Pointer)(ptr), fl}
}

var selectHelper = js.Global.Get("$select").Interface().(func(...any) *js.Object)

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

func deepValueEqualJs(v1, v2 Value, visited [][2]unsafe.Pointer) bool {
	if !v1.IsValid() || !v2.IsValid() {
		return !v1.IsValid() && !v2.IsValid()
	}
	if v1.Type() != v2.Type() {
		return false
	}
	if v1.Type() == jsObjectPtr {
		return unwrapJsObject(jsObjectPtr, v1.object()) == unwrapJsObject(jsObjectPtr, v2.object())
	}

	switch v1.Kind() {
	case abi.Array, abi.Map, abi.Slice, abi.Struct:
		for _, entry := range visited {
			if v1.ptr == entry[0] && v2.ptr == entry[1] {
				return true
			}
		}
		visited = append(visited, [2]unsafe.Pointer{v1.ptr, v2.ptr})
	}

	switch v1.Kind() {
	case abi.Array, abi.Slice:
		if v1.Kind() == abi.Slice {
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
	case abi.Interface:
		if v1.IsNil() || v2.IsNil() {
			return v1.IsNil() && v2.IsNil()
		}
		return deepValueEqualJs(v1.Elem(), v2.Elem(), visited)
	case abi.Pointer:
		return deepValueEqualJs(v1.Elem(), v2.Elem(), visited)
	case abi.Struct:
		n := v1.NumField()
		for i := 0; i < n; i++ {
			if !deepValueEqualJs(v1.Field(i), v2.Field(i), visited) {
				return false
			}
		}
		return true
	case abi.Map:
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
	case abi.Func:
		return v1.IsNil() && v2.IsNil()
	case abi.UnsafePointer:
		return v1.object() == v2.object()
	}

	return js.Global.Call("$interfaceIsEqual", js.InternalObject(valueInterface(v1)), js.InternalObject(valueInterface(v2))).Bool()
}
