//go:build js

package reflect

import (
	"strconv"
	"unsafe"

	"internal/abi"
	"internal/itoa"

	"github.com/gopherjs/gopherjs/js"
)

func init() {
	// avoid dead code elimination
	used := func(i any) {}
	used(rtype{})
	used(uncommonType{})
	used(arrayType{})
	used(chanType{})
	used(funcType{})
	used(interfaceType{})
	used(mapType{})
	used(ptrType{})
	used(sliceType{})
	used(structType{})
	used(structField{})
	used(toKindTypeExt)
}

// New returns a Value representing a pointer to a new zero value
// for the specified type. That is, the returned Value's Type is PtrTo(typ).
//
// The upstream version includes an extra check to avoid creating types that
// are tagged as go:notinheap. This shouldn't matter in GopherJS, and tracking
// that state is over-complex, so we just skip that check.
//
//gopherjs:replace
func New(typ Type) Value {
	if typ == nil {
		panic("reflect: New(nil)")
	}
	t := toAbiType(typ)
	pt := t.PtrTo()
	ptr := unsafe_New(t)
	fl := flag(Pointer)
	return Value{pt, ptr, fl}
}

//gopherjs:new
func jsType(typ Type) *js.Object {
	return toAbiType(typ).JsType()
}

//gopherjs:new
func toAbiType(typ Type) *abi.Type {
	return typ.(*rtype).common()
}

//gopherjs:replace
func toRType(t *abi.Type) *rtype {
	rtyp := &rtype{}
	// Assign t to the abiType. The abiType is a `*Type` and the t
	// field on `rtype` is `Type`. However, this is valid because of how
	// pointers and references work in JS. We set this so that the t
	// isn't a copy but the actual abiType object.
	js.InternalObject(rtyp).Set("t", js.InternalObject(t))
	return rtyp
}

//gopherjs:purge
func addReflectOff(ptr unsafe.Pointer) int32

//gopherjs:replace
func (t *rtype) nameOff(off aNameOff) abi.Name {
	return toAbiType(t).NameOff(off)
}

//gopherjs:replace
func resolveReflectName(n abi.Name) aNameOff {
	return abi.ResolveReflectName(n)
}

//gopherjs:replace
func (t *rtype) typeOff(off aTypeOff) *abi.Type {
	return toAbiType(t).TypeOff(off)
}

//gopherjs:replace
func resolveReflectType(t *abi.Type) aTypeOff {
	return abi.ResolveReflectType(t)
}

//gopherjs:replace
func (t *rtype) textOff(off aTextOff) unsafe.Pointer {
	return toAbiType(t).TextOff(off)
}

//gopherjs:replace
func resolveReflectText(ptr unsafe.Pointer) aTextOff {
	return abi.ResolveReflectText(ptr)
}

//gopherjd:replace
func pkgPath(n abi.Name) string {
	return n.PkgPath()
}

//gopherjs:new
func makeValue(t *abi.Type, v *js.Object, fl flag) Value {
	switch t.Kind() {
	case abi.Array, abi.Struct, abi.Pointer:
		return Value{t, unsafe.Pointer(v.Unsafe()), fl | flag(t.Kind())}
	}
	return Value{t, unsafe.Pointer(js.Global.Call("$newDataPointer", v, t.JsPtrTo()).Unsafe()), fl | flag(t.Kind()) | flagIndir}
}

//gopherjs:replace
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

	return makeValue(toAbiType(typ), js.Global.Call("$makeSlice", jsType(typ), len, cap, js.InternalObject(func() *js.Object { return jsType(typ.Elem()).Call("zero") })), 0)
}

//gopherjs:replace
func TypeOf(i any) Type {
	if i == nil {
		return nil
	}
	return toRType(rtypeOf(i))
}

//gopherjs:replace
func ValueOf(i any) Value {
	if i == nil {
		return Value{}
	}
	return makeValue(rtypeOf(i), js.InternalObject(i).Get("$val"), 0)
}

//gopherjs:replace
func ArrayOf(count int, elem Type) Type {
	if count < 0 {
		panic("reflect: negative length passed to ArrayOf")
	}

	return toRType(abi.ReflectType(js.Global.Call("$arrayType", jsType(elem), count)))
}

//gopherjs:replace
func ChanOf(dir ChanDir, t Type) Type {
	return toRType(abi.ReflectType(js.Global.Call("$chanType", jsType(t), dir == SendDir, dir == RecvDir)))
}

//gopherjs:replace
func FuncOf(in, out []Type, variadic bool) Type {
	if variadic && (len(in) == 0 || in[len(in)-1].Kind() != Slice) {
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
	return toRType(abi.ReflectType(js.Global.Call("$funcType", jsIn, jsOut, variadic)))
}

//gopherjs:replace
func MapOf(key, elem Type) Type {
	switch key.Kind() {
	case Func, Map, Slice:
		panic("reflect.MapOf: invalid key type " + key.String())
	}

	return toRType(abi.ReflectType(js.Global.Call("$mapType", jsType(key), jsType(elem))))
}

//gopherjs:replace
func (t *rtype) ptrTo() *abi.Type {
	return abi.ReflectType(js.Global.Call("$ptrType", jsType(t)))
}

//gopherjs:replace
func SliceOf(t Type) Type {
	return toRType(abi.ReflectType(js.Global.Call("$sliceType", jsType(t))))
}

//gopherjs:replace
func StructOf(fields []StructField) Type {
	var (
		jsFields  = make([]*js.Object, len(fields))
		fset      = map[string]struct{}{}
		pkgpath   string
		hasGCProg bool
	)
	for i, field := range fields {
		if field.Name == "" {
			panic("reflect.StructOf: field " + strconv.Itoa(i) + " has no name")
		}
		if !isValidFieldName(field.Name) {
			panic("reflect.StructOf: field " + strconv.Itoa(i) + " has invalid name")
		}
		if field.Type == nil {
			panic("reflect.StructOf: field " + strconv.Itoa(i) + " has no type")
		}
		f, fpkgpath := runtimeStructField(field)
		ft := f.Typ
		if ft.Kind()&kindGCProg != 0 {
			hasGCProg = true
		}
		if fpkgpath != "" {
			if pkgpath == "" {
				pkgpath = fpkgpath
			} else if pkgpath != fpkgpath {
				panic("reflect.Struct: fields with different PkgPath " + pkgpath + " and " + fpkgpath)
			}
		}
		name := field.Name
		if f.Embedded() {
			// Embedded field
			if field.Type.Kind() == Ptr {
				// Embedded ** and *interface{} are illegal
				elem := field.Type.Elem()
				if k := elem.Kind(); k == Ptr || k == Interface {
					panic("reflect.StructOf: illegal anonymous field type " + field.Type.String())
				}
			}
			switch field.Type.Kind() {
			case Interface:
			case Ptr:
				ptr := (*ptrType)(unsafe.Pointer(ft))
				if unt := ptr.Uncommon(); unt != nil {
					if i > 0 && unt.Mcount > 0 {
						// Issue 15924.
						panic("reflect: embedded type with methods not implemented if type is not first field")
					}
					if len(fields) > 1 {
						panic("reflect: embedded type with methods not implemented if there is more than one field")
					}
				}
			default:
				if unt := ft.Uncommon(); unt != nil {
					if i > 0 && unt.Mcount > 0 {
						// Issue 15924.
						panic("reflect: embedded type with methods not implemented if type is not first field")
					}
					if len(fields) > 1 && ft.Kind()&kindDirectIface != 0 {
						panic("reflect: embedded type with methods not implemented for non-pointer type")
					}
				}
			}
		}

		if _, dup := fset[name]; dup && name != "_" {
			panic("reflect.StructOf: duplicate field " + name)
		}
		fset[name] = struct{}{}
		// To be consistent with Compiler's behavior we need to avoid externalizing
		// the "name" property. The line below is effectively an inverse of the
		// internalStr() function.
		jsf := js.InternalObject(struct{ name string }{name})
		// The rest is set through the js.Object() interface, which the compiler will
		// externalize for us.
		jsf.Set("prop", name)
		jsf.Set("exported", f.Name.IsExported())
		jsf.Set("typ", jsType(field.Type))
		jsf.Set("tag", field.Tag)
		jsf.Set("embedded", field.Anonymous)
		jsFields[i] = jsf
	}
	_ = hasGCProg
	typ := js.Global.Call("$structType", "", jsFields)
	if pkgpath != "" {
		typ.Set("pkgPath", pkgpath)
	}
	return toRType(abi.ReflectType(typ))
}

//gopherjs:replace
func Zero(typ Type) Value {
	return makeValue(toAbiType(typ), jsType(typ).Call("zero"), 0)
}

//gopherjs:replace
func unsafe_New(typ *abi.Type) unsafe.Pointer {
	return abi.UnsafeNew(typ)
}

//gopherjs:replace
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

//gopherjs:replace
func MakeFunc(typ Type, fn func(args []Value) (results []Value)) Value {
	if typ.Kind() != Func {
		panic("reflect: call of MakeFunc with non-Func type")
	}

	t := typ.common()
	ftyp := (*funcType)(unsafe.Pointer(t))

	fv := js.MakeFunc(func(this *js.Object, arguments []*js.Object) any {
		// Convert raw JS arguments into []Value the user-supplied function expects.
		args := make([]Value, ftyp.NumIn())
		for i := range args {
			argType := ftyp.In(i)
			args[i] = makeValue(argType, arguments[i], 0)
		}

		// Call the user-supplied function.
		resultsSlice := fn(args)

		// Verify that returned value types are compatible with the function type specified by the caller.
		if want, got := ftyp.NumOut(), len(resultsSlice); want != got {
			panic("reflect: expected " + strconv.Itoa(want) + " return values, got " + strconv.Itoa(got))
		}
		for i, rtyp := range ftyp.OutSlice() {
			if !resultsSlice[i].Type().AssignableTo(toRType(rtyp)) {
				panic("reflect: " + strconv.Itoa(i) + " return value type is not compatible with the function declaration")
			}
		}

		// Rearrange return values according to the expected function signature.
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

//gopherjs:replace
func typedmemmove(t *abi.Type, dst, src unsafe.Pointer) {
	abi.TypedMemMove(t, dst, src)
}

//gopherjs:replace
func makechan(typ *abi.Type, size int) (ch unsafe.Pointer) {
	ctyp := (*chanType)(unsafe.Pointer(typ))
	return unsafe.Pointer(js.Global.Get("$Chan").New(ctyp.Elem.JsType(), size).Unsafe())
}

//gopherjs:replace
func makemap(t *abi.Type, cap int) (m unsafe.Pointer) {
	return unsafe.Pointer(js.Global.Get("Map").New().Unsafe())
}

//gopherjs:new
func keyFor(t *abi.Type, key unsafe.Pointer) (*js.Object, *js.Object) {
	kv := js.InternalObject(key)
	if kv.Get("$get") != js.Undefined {
		kv = kv.Call("$get")
	}
	k := t.Key().JsType().Call("keyFor", kv)
	return kv, k
}

//gopherjs:replace
func mapaccess(t *abi.Type, m, key unsafe.Pointer) unsafe.Pointer {
	if !js.InternalObject(m).Bool() {
		return nil // nil map
	}
	_, k := keyFor(t, key)
	entry := js.InternalObject(m).Call("get", k)
	if entry == js.Undefined {
		return nil
	}
	return unsafe.Pointer(js.Global.Call("$newDataPointer", entry.Get("v"), t.Elem().JsPtrTo()).Unsafe())
}

//gopherjs:replace
func mapassign(t *abi.Type, m, key, val unsafe.Pointer) {
	kv, k := keyFor(t, key)
	jsVal := js.InternalObject(val).Call("$get")
	et := t.Elem()
	if et.Kind() == abi.Struct {
		newVal := et.JsType().Call("zero")
		abi.CopyStruct(newVal, jsVal, et)
		jsVal = newVal
	}
	entry := js.Global.Get("Object").New()
	entry.Set("k", kv)
	entry.Set("v", jsVal)
	js.InternalObject(m).Call("set", k, entry)
}

//gopherjs:replace
func mapdelete(t *abi.Type, m unsafe.Pointer, key unsafe.Pointer) {
	_, k := keyFor(t, key)
	if !js.InternalObject(m).Bool() {
		return // nil map
	}
	js.InternalObject(m).Call("delete", k)
}

// TODO(nevkonatkte): The following three "faststr" implementations are meant to
// perform better for the common case of string-keyed maps (see upstream:
// https://github.com/golang/go/commit/23832ba2e2fb396cda1dacf3e8afcb38ec36dcba)
// However, the stubs below will perform the same or worse because of the extra
// string-to-pointer conversion. Not sure how to fix this without significant
// code duplication, however.

//gopherjs:replace
func mapaccess_faststr(t *abi.Type, m unsafe.Pointer, key string) (val unsafe.Pointer) {
	return mapaccess(t, m, unsafe.Pointer(&key))
}

//gopherjs:replace
func mapassign_faststr(t *abi.Type, m unsafe.Pointer, key string, val unsafe.Pointer) {
	mapassign(t, m, unsafe.Pointer(&key), val)
}

//gopherjs:replace
func mapdelete_faststr(t *abi.Type, m unsafe.Pointer, key string) {
	mapdelete(t, m, unsafe.Pointer(&key))
}

//gopherjs:replace
type hiter struct {
	t    *abi.Type
	m    *js.Object // Underlying map object.
	keys *js.Object
	i    int

	// last is the last object the iterator indicates. If this object exists, the
	// functions that return the current key or value returns this object,
	// regardless of the current iterator. It is because the current iterator
	// might be stale due to key deletion in a loop.
	last *js.Object
}

//gopherjs:new
func (iter *hiter) skipUntilValidKey() {
	for iter.i < iter.keys.Length() {
		k := iter.keys.Index(iter.i)
		entry := iter.m.Call("get", k)
		if entry != js.Undefined {
			break
		}
		// The key is already deleted. Move on the next item.
		iter.i++
	}
}

//gopherjs:replace
func mapiterinit(t *abi.Type, m unsafe.Pointer, it *hiter) {
	mapObj := js.InternalObject(m)
	keys := js.Global.Get("Array").New()
	if mapObj.Get("keys") != js.Undefined {
		keysIter := mapObj.Call("keys")
		if mapObj.Get("keys") != js.Undefined {
			keys = js.Global.Get("Array").Call("from", keysIter)
		}
	}

	*it = hiter{
		t:    t,
		m:    mapObj,
		keys: keys,
		i:    0,
		last: nil,
	}
}

//gopherjs:replace
func mapiterkey(it *hiter) unsafe.Pointer {
	var kv *js.Object
	if it.last != nil {
		kv = it.last
	} else {
		it.skipUntilValidKey()
		if it.i == it.keys.Length() {
			return nil
		}
		k := it.keys.Index(it.i)
		kv = it.m.Call("get", k)

		// Record the key-value pair for later accesses.
		it.last = kv
	}
	return unsafe.Pointer(js.Global.Call("$newDataPointer", kv.Get("k"), it.t.Key().JsPtrTo()).Unsafe())
}

//gopherjs:replace
func mapiterelem(it *hiter) unsafe.Pointer {
	var kv *js.Object
	if it.last != nil {
		kv = it.last
	} else {
		it.skipUntilValidKey()
		if it.i == it.keys.Length() {
			return nil
		}
		k := it.keys.Index(it.i)
		kv = it.m.Call("get", k)
		it.last = kv
	}
	return unsafe.Pointer(js.Global.Call("$newDataPointer", kv.Get("v"), it.t.Elem().JsPtrTo()).Unsafe())
}

//gopherjs:replace
func mapiternext(it *hiter) {
	it.last = nil
	it.i++
}

//gopherjs:replace
func maplen(m unsafe.Pointer) int {
	return js.InternalObject(m).Get("size").Int()
}

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
		panic(&ValueError{"reflect.Convert", k})
	}
	return Value{typ.common(), unsafe.Pointer(val.Unsafe()), v.flag.ro() | v.flag&flagIndir | flag(typ.Kind())}
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
	return Value{t.common(), unsafe.Pointer(array.Unsafe()), v.flag&^(flagIndir|flagAddr|flagKindMask) | flag(Ptr)}
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
	return Value{t.common(), unsafe.Pointer(arr.Unsafe()), v.flag&^(flagAddr|flagKindMask) | flag(Array)}
}

//gopherjs:replace
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
	var stringCopy bool
	if sk != Array && sk != Slice {
		stringCopy = sk == String && dst.typ().Elem().Kind() == abi.Uint8
		if !stringCopy {
			panic(&ValueError{"reflect.Copy", sk})
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
func methodReceiver(op string, v Value, methodIndex int) (rcvrtype *abi.Type, t *funcType, fn unsafe.Pointer) {
	i := methodIndex
	var prop string
	if v.typ().Kind() == abi.Interface {
		tt := (*interfaceType)(unsafe.Pointer(v.typ()))
		if i < 0 || i >= len(tt.Methods) {
			panic("reflect: internal error: invalid method index")
		}
		m := &tt.Methods[i]
		if !tt.NameOff(m.Name).IsExported() {
			panic("reflect: " + op + " of unexported method")
		}
		t = (*funcType)(unsafe.Pointer(tt.typeOff(m.Typ)))
		prop = tt.NameOff(m.Name).Name()
	} else {
		rcvrtype = v.typ()
		ms := v.typ().ExportedMethods()
		if uint(i) >= uint(len(ms)) {
			panic("reflect: internal error: invalid method index")
		}
		m := ms[i]
		if !v.typ().NameOff(m.Name).IsExported() {
			panic("reflect: " + op + " of unexported method")
		}
		t = (*funcType)(unsafe.Pointer(v.typ().TypeOff(m.Mtyp)))
		prop = js.Global.Call("$methodSet", v.typ().JsType()).Index(i).Get("prop").String()
	}
	rcvr := v.object()
	if v.typ().IsWrapped() {
		rcvr = v.typ().JsType().New(rcvr)
	}
	fn = unsafe.Pointer(rcvr.Get(prop).Unsafe())
	return
}

//gopherjs:replace
func valueInterface(v Value, safe bool) any {
	if v.flag == 0 {
		panic(&ValueError{"reflect.Value.Interface", 0})
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

//gopherjs:replace
func ifaceE2I(t *abi.Type, src any, dst unsafe.Pointer) {
	abi.IfaceE2I(t, src, dst)
}

//gopherjs:replace
func makeMethodValue(op string, v Value) Value {
	if v.flag&flagMethod == 0 {
		panic("reflect: internal error: invalid use of makePartialFunc")
	}

	_, _, fn := methodReceiver(op, v, int(v.flag)>>flagMethodShift)
	rcvr := v.object()
	if v.typ().IsWrapped() {
		rcvr = v.typ().JsType().New(rcvr)
	}
	fv := js.MakeFunc(func(this *js.Object, arguments []*js.Object) any {
		return js.InternalObject(fn).Call("apply", rcvr, arguments)
	})
	return Value{
		typ_: v.Type().common(),
		ptr:  unsafe.Pointer(fv.Unsafe()),
		flag: v.flag.ro() | flag(Func),
	}
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

//gopherjs:replace
func (t *rtype) Method(i int) (m Method) {
	if t.Kind() == Interface {
		tt := (*interfaceType)(unsafe.Pointer(t))
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
