//go:build js

package reflect

import (
	"errors"
	"strconv"
	"unsafe"

	"internal/abi"

	"github.com/gopherjs/gopherjs/compiler/natives/src/internal/unsafeheader"
	"github.com/gopherjs/gopherjs/js"
)

//gopherjs:purge This is the header for an any interface and invalid for GopherJS.
type emptyInterface struct{}

//gopherjs:purge This is the header for an interface value with methods and invalid for GopherJS.
type nonEmptyInterface struct{}

//gopherjs:purge
func packEface(v Value) any

//gopherjs:purge
func unpackEface(i any) Value

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

//gopherjs:replace
func ValueOf(i any) Value {
	if i == nil {
		return Value{}
	}
	return makeValue(abi.ReflectType(js.InternalObject(i).Get("constructor")), js.InternalObject(i).Get("$val"), 0)
}

//gopherjs:replace
func unsafe_New(typ *abi.Type) unsafe.Pointer {
	return abi.UnsafeNew(typ)
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
func Zero(typ Type) Value {
	return makeValue(toAbiType(typ), jsType(typ).Call("zero"), 0)
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

//gopherjs:purge
func storeRcvr(v Value, p unsafe.Pointer)

//gopherjs:purge
func callMethod(ctxt *methodValue, frame unsafe.Pointer, retValid *bool, regs *abi.RegArgs)

//gopherjs:replace
func MakeFunc(typ Type, fn func(args []Value) (results []Value)) Value {
	if typ.Kind() != Func {
		panic("reflect: call of MakeFunc with non-Func type")
	}

	t := typ.common()
	ftyp := t.FuncType()

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
func ifaceE2I(t *abi.Type, src any, dst unsafe.Pointer) {
	abi.IfaceE2I(t, src, dst)
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
func (v Value) object() *js.Object {
	if v.typ().Kind() == abi.Array || v.typ().Kind() == abi.Struct {
		return js.InternalObject(v.ptr)
	}
	jsTyp := v.typ().JsType()
	if v.flag&flagIndir != 0 {
		val := js.InternalObject(v.ptr).Call("$get")
		if val != js.Global.Get("$ifaceNil") && val.Get("constructor") != jsTyp {
			switch v.typ().Kind() {
			case abi.Uint64, abi.Int64:
				val = jsTyp.New(val.Get("$high"), val.Get("$low"))
			case abi.Complex64, abi.Complex128:
				val = jsTyp.New(val.Get("$real"), val.Get("$imag"))
			case abi.Slice:
				if val == val.Get("constructor").Get("nil") {
					val = jsTyp.Get("nil")
					break
				}
				newVal := jsTyp.New(val.Get("$array"))
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
	case directlyAssignable(dst, v.typ()):
		// Overwrite type so that they match.
		// Same memory layout, so no harm done.
		fl := v.flag&(flagAddr|flagIndir) | v.flag.ro()
		fl |= flag(dst.Kind())
		return Value{dst, v.ptr, fl}

	case implements(dst, v.typ()):
		if target == nil {
			target = unsafe_New(dst)
		}
		// GopherJS: Skip the v.Kind() == Interface && v.IsNil() if statement
		//           from upstream. ifaceE2I below does not panic, and it needs
		//           to run, given its custom implementation.
		x := valueInterface(v, false)
		if dst.NumMethod() == 0 {
			*(*any)(target) = x
		} else {
			ifaceE2I(dst, x, target)
		}
		return Value{dst, target, flagIndir | flag(Interface)}
	}

	// Failed.
	panic(context + ": value of type " + v.typ().String() + " is not assignable to type " + dst.String())
}

//gopherjs:new
var callHelper = js.Global.Get("$call").Interface().(func(...any) *js.Object)

//gopherjs:replace
func (v Value) call(op string, in []Value) []Value {
	var (
		t    *funcType
		fn   unsafe.Pointer
		rcvr *js.Object
	)
	if v.flag&flagMethod != 0 {
		_, t, fn = methodReceiver(op, v, int(v.flag)>>flagMethodShift)
		rcvr = v.object()
		if v.typ().IsWrapped() {
			rcvr = v.typ().JsType().New(rcvr)
		}
	} else {
		t = v.typ().FuncType()
		fn = unsafe.Pointer(v.object().Unsafe())
		rcvr = js.Undefined
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
		if xt, targ := in[i].Type(), toRType(t.In(i)); !xt.AssignableTo(targ) {
			panic("reflect: " + op + " using " + xt.String() + " as type " + targ.String())
		}
	}
	if !isSlice && t.IsVariadic() {
		// prepare slice for remaining values
		m := len(in) - n
		slice := MakeSlice(toRType(t.In(n)), m, m)
		elem := toRType(t.In(n).Elem())
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
		argsArray.SetIndex(i, abi.UnwrapJsObject(t.In(i), arg.assignTo("reflect.Value.Call", t.In(i), nil).object()))
	}
	results := callHelper(js.InternalObject(fn), rcvr, argsArray)

	switch nout {
	case 0:
		return nil
	case 1:
		return []Value{makeValue(t.Out(0), abi.WrapJsObject(t.Out(0), results), 0)}
	default:
		ret := make([]Value, nout)
		for i := range ret {
			ret[i] = makeValue(t.Out(i), abi.WrapJsObject(t.Out(i), results.Index(i)), 0)
		}
		return ret
	}
}

//gopherjs:replace Used a pointer cast to get *rtype that doesn't work in JS.
func (v Value) Type() Type {
	if v.flag != 0 && v.flag&flagMethod == 0 {
		return toRType(v.typ_)
	}
	return v.typeSlow()
}

//gopherjs:replace
func (v Value) Cap() int {
	k := v.kind()
	switch k {
	case Array:
		return v.typ().Len()
	case Chan, Slice:
		return v.object().Get("$capacity").Int()
	case Ptr:
		if v.typ().Elem().Kind() == abi.Array {
			return v.typ().Elem().Len()
		}
		panic("reflect: call of reflect.Value.Cap on ptr to non-array Value")
	}
	panic(&ValueError{"reflect.Value.Cap", k})
}

//gopherjs:replace
func (v Value) Elem() Value {
	switch k := v.kind(); k {
	case Interface:
		val := v.object()
		if val == js.Global.Get("$ifaceNil") {
			return Value{}
		}
		typ := abi.ReflectType(val.Get("constructor"))
		return makeValue(typ, val.Get("$val"), v.flag.ro())

	case Ptr:
		if v.IsNil() {
			return Value{}
		}
		val := v.object()
		tt := (*ptrType)(unsafe.Pointer(v.typ()))
		fl := v.flag&flagRO | flagIndir | flagAddr
		fl |= flag(tt.Elem.Kind())
		return Value{tt.Elem, unsafe.Pointer(abi.WrapJsObject(tt.Elem, val).Unsafe()), fl}

	default:
		panic(&ValueError{"reflect.Value.Elem", k})
	}
}

//gopherjs:replace
func (v Value) Field(i int) Value {
	if v.kind() != Struct {
		panic(&ValueError{"reflect.Value.Field", v.kind()})
	}
	tt := (*structType)(unsafe.Pointer(v.typ()))
	if uint(i) >= uint(len(tt.Fields)) {
		panic("reflect: Field index out of range")
	}

	prop := v.typ().JsType().Get("fields").Index(i).Get("prop").String()
	field := &tt.Fields[i]
	typ := field.Typ

	fl := v.flag&(flagStickyRO|flagIndir|flagAddr) | flag(typ.Kind())
	if !field.Name.IsExported() {
		if field.Embedded() {
			fl |= flagEmbedRO
		} else {
			fl |= flagStickyRO
		}
	}

	if tag := tt.Fields[i].Name.Tag(); tag != "" && i != 0 {
		if jsTag := abi.GetJsTag(tag); jsTag != "" {
			for {
				v = v.Field(0)
				if v.typ() == abi.JsObjectPtr {
					o := v.object().Get("object")
					return Value{typ, unsafe.Pointer(typ.JsPtrTo().New(
						js.InternalObject(func() *js.Object { return js.Global.Call("$internalize", o.Get(jsTag), typ.JsType()) }),
						js.InternalObject(func(x *js.Object) { o.Set(jsTag, js.Global.Call("$externalize", x, typ.JsType())) }),
					).Unsafe()), fl}
				}
				if v.typ().Kind() == abi.Pointer {
					v = v.Elem()
				}
			}
		}
	}

	s := js.InternalObject(v.ptr)
	if fl&flagIndir != 0 && typ.Kind() != abi.Array && typ.Kind() != abi.Struct {
		return Value{typ, unsafe.Pointer(typ.JsPtrTo().New(
			js.InternalObject(func() *js.Object { return abi.WrapJsObject(typ, s.Get(prop)) }),
			js.InternalObject(func(x *js.Object) { s.Set(prop, abi.UnwrapJsObject(typ, x)) }),
		).Unsafe()), fl}
	}
	return makeValue(typ, abi.WrapJsObject(typ, s.Get(prop)), fl)
}

//gopherjs:replace
func (v Value) UnsafePointer() unsafe.Pointer {
	return unsafe.Pointer(v.Pointer())
}

//gopherjs:replace
func (v Value) grow(n int) {
	if n < 0 {
		panic(`reflect.Value.Grow: negative len`)
	}

	s := v.object()
	len := s.Get(`$length`).Int()
	if len+n < 0 {
		panic(`reflect.Value.Grow: slice overflow`)
	}

	cap := s.Get(`$capacity`).Int()
	if len+n > cap {
		ns := js.Global.Call("$growSlice", s, len+n)
		js.InternalObject(v.ptr).Call("$set", ns)
	}
}

// extendSlice is used by native reflect.Append and reflect.AppendSlice
// Overridden to avoid the use of `unsafeheader.Slice` since GopherJS
// uses different slice implementation.
//
//gopherjs:replace
func (v Value) extendSlice(n int) Value {
	v.mustBeExported()
	v.mustBe(Slice)

	s := v.object()
	sNil := v.typ().JsType().Get(`nil`)
	fl := flagIndir | flag(Slice)
	if s == sNil && n <= 0 {
		return makeValue(v.typ(), abi.WrapJsObject(v.typ(), sNil), fl)
	}

	newSlice := v.typ().JsType().New(s.Get("$array"))
	newSlice.Set("$offset", s.Get("$offset"))
	newSlice.Set("$length", s.Get("$length"))
	newSlice.Set("$capacity", s.Get("$capacity"))

	v2 := makeValue(v.typ(), abi.WrapJsObject(v.typ(), newSlice), fl)
	v2.grow(n)
	s2 := v2.object()
	s2.Set(`$length`, s2.Get(`$length`).Int()+n)
	return v2
}

//gopherjs:purge
func mapclear(t *abi.Type, m unsafe.Pointer)

//gopherjs:purge
func typedarrayclear(elemType *abi.Type, ptr unsafe.Pointer, len int)

// TODO(grantnelson-wf): Make sure this is tested since it is new.
//
//gopherjs:replace
func (v Value) Clear() {
	switch v.Kind() {
	case Slice:
		st := (*sliceType)(unsafe.Pointer(v.typ()))
		elem := st.Elem
		zeroFn := elem.JsType().Get("zero")
		a := js.InternalObject(v.ptr)
		offset := a.Get("$offset").Int()
		length := a.Get("$length").Int()
		for i := 0; i < length; i++ {
			a.SetIndex(i+offset, zeroFn.Invoke())
		}
	// case Map:
	// TODO(grantnelson-wf): Finish implementing
	// mapclear(v.typ(), v.pointer())
	default:
		panic(&ValueError{"reflect.Value.Clear", v.Kind()})
	}
}

//gopherjs:replace
func (v Value) Index(i int) Value {
	switch k := v.kind(); k {
	case Array:
		tt := v.typ().ArrayType()
		if i < 0 || i > int(tt.Len) {
			panic("reflect: array index out of range")
		}
		typ := tt.Elem
		fl := v.flag&(flagIndir|flagAddr) | v.flag.ro() | flag(typ.Kind())

		a := js.InternalObject(v.ptr)
		if fl&flagIndir != 0 && typ.Kind() != abi.Array && typ.Kind() != abi.Struct {
			return Value{typ, unsafe.Pointer(typ.JsPtrTo().New(
				js.InternalObject(func() *js.Object { return abi.WrapJsObject(typ, a.Index(i)) }),
				js.InternalObject(func(x *js.Object) { a.SetIndex(i, abi.UnwrapJsObject(typ, x)) }),
			).Unsafe()), fl}
		}
		return makeValue(typ, abi.WrapJsObject(typ, a.Index(i)), fl)

	case Slice:
		s := v.object()
		if i < 0 || i >= s.Get("$length").Int() {
			panic("reflect: slice index out of range")
		}
		tt := (*sliceType)(unsafe.Pointer(v.typ()))
		typ := tt.Elem
		fl := flagAddr | flagIndir | v.flag.ro() | flag(typ.Kind())

		i += s.Get("$offset").Int()
		a := s.Get("$array")
		if fl&flagIndir != 0 && typ.Kind() != abi.Array && typ.Kind() != abi.Struct {
			return Value{typ, unsafe.Pointer(typ.JsPtrTo().New(
				js.InternalObject(func() *js.Object { return abi.WrapJsObject(typ, a.Index(i)) }),
				js.InternalObject(func(x *js.Object) { a.SetIndex(i, abi.UnwrapJsObject(typ, x)) }),
			).Unsafe()), fl}
		}
		return makeValue(typ, abi.WrapJsObject(typ, a.Index(i)), fl)

	case String:
		str := *(*string)(v.ptr)
		if i < 0 || i >= len(str) {
			panic("reflect: string index out of range")
		}
		fl := v.flag.ro() | flag(Uint8) | flagIndir
		c := str[i]
		return Value{uint8Type, unsafe.Pointer(&c), fl}

	default:
		panic(&ValueError{"reflect.Value.Index", k})
	}
}

//gopherjs:replace
func (v Value) InterfaceData() [2]uintptr {
	panic(errors.New("InterfaceData is not supported by GopherJS"))
}

//gopherjs:replace
func (v Value) SetZero() {
	v.mustBeAssignable()
	v.Set(Zero(toRType(v.typ())))
}

//gopherjs:replace
func (v Value) IsNil() bool {
	switch k := v.kind(); k {
	case Ptr, Slice:
		return v.object() == v.typ().JsType().Get("nil")
	case Chan:
		return v.object() == js.Global.Get("$chanNil")
	case Func:
		return v.object() == js.Global.Get("$throwNilPointerError")
	case Map:
		return v.object() == js.InternalObject(false)
	case Interface:
		return v.object() == js.Global.Get("$ifaceNil")
	case UnsafePointer:
		return v.object().Unsafe() == 0
	default:
		panic(&ValueError{"reflect.Value.IsNil", k})
	}
}

//gopherjs:replace
func (v Value) Len() int {
	switch k := v.kind(); k {
	case Array, String:
		return v.object().Length()
	case Slice:
		return v.object().Get("$length").Int()
	case Chan:
		return v.object().Get("$buffer").Get("length").Int()
	case Map:
		return v.object().Get("size").Int()
	case Ptr:
		if elem := v.typ().Elem(); elem.Kind() == abi.Array {
			return elem.Len()
		}
		panic("reflect: call of reflect.Value.Len on ptr to non-array Value")
	default:
		panic(&ValueError{"reflect.Value.Len", k})
	}
}

//gopherjs:purge Not used since Len() is overridden.
func (v Value) lenNonSlice() int

//gopherjs:replace
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

//gopherjs:replace
func (v Value) Set(x Value) {
	v.mustBeAssignable()
	x.mustBeExported()
	x = x.assignTo("reflect.Set", v.typ(), nil)
	if v.flag&flagIndir != 0 {
		switch v.typ().Kind() {
		case abi.Array, abi.Struct:
			v.typ().JsType().Call("copy", js.InternalObject(v.ptr), js.InternalObject(x.ptr))
		case abi.Interface:
			js.InternalObject(v.ptr).Call("$set", js.InternalObject(valueInterface(x, false)))
		default:
			js.InternalObject(v.ptr).Call("$set", x.object())
		}
		return
	}
	v.ptr = x.ptr
}

//gopherjs:replace
func (v Value) bytesSlow() []byte {
	switch v.kind() {
	case Slice:
		if v.typ().Elem().Kind() != abi.Uint8 {
			panic("reflect.Value.Bytes of non-byte slice")
		}
		return *(*[]byte)(v.ptr)
	case Array:
		if v.typ().Elem().Kind() != abi.Uint8 {
			panic("reflect.Value.Bytes of non-byte array")
		}
		if !v.CanAddr() {
			panic("reflect.Value.Bytes of unaddressable byte array")
		}
		// GOPHERJS: Replace the following with JS to avoid using unsafe pointers.
		//   p := (*byte)(v.ptr)
		//   n := int((*arrayType)(unsafe.Pointer(v.typ)).len)
		//   return unsafe.Slice(p, n)
		return js.InternalObject(v.ptr).Interface().([]byte)
	}
	panic(&ValueError{"reflect.Value.Bytes", v.kind()})
}

//gopherjs:replace
func (v Value) SetBytes(x []byte) {
	v.mustBeAssignable()
	v.mustBe(Slice)
	if v.typ().Elem().Kind() != abi.Uint8 {
		panic("reflect.Value.SetBytes of non-byte slice")
	}
	slice := js.InternalObject(x)
	if toRType(v.typ()).Name() != "" || toRType(v.typ()).Elem().Name() != "" {
		typedSlice := v.typ().JsType().New(slice.Get("$array"))
		typedSlice.Set("$offset", slice.Get("$offset"))
		typedSlice.Set("$length", slice.Get("$length"))
		typedSlice.Set("$capacity", slice.Get("$capacity"))
		slice = typedSlice
	}
	js.InternalObject(v.ptr).Call("$set", slice)
}

//gopherjs:replace
func (v Value) SetCap(n int) {
	v.mustBeAssignable()
	v.mustBe(Slice)
	s := js.InternalObject(v.ptr).Call("$get")
	if n < s.Get("$length").Int() || n > s.Get("$capacity").Int() {
		panic("reflect: slice capacity out of range in SetCap")
	}
	newSlice := v.typ().JsType().New(s.Get("$array"))
	newSlice.Set("$offset", s.Get("$offset"))
	newSlice.Set("$length", s.Get("$length"))
	newSlice.Set("$capacity", n)
	js.InternalObject(v.ptr).Call("$set", newSlice)
}

//gopherjs:replace
func (v Value) SetLen(n int) {
	v.mustBeAssignable()
	v.mustBe(Slice)
	s := js.InternalObject(v.ptr).Call("$get")
	if n < 0 || n > s.Get("$capacity").Int() {
		panic("reflect: slice length out of range in SetLen")
	}
	newSlice := v.typ().JsType().New(s.Get("$array"))
	newSlice.Set("$offset", s.Get("$offset"))
	newSlice.Set("$length", n)
	newSlice.Set("$capacity", s.Get("$capacity"))
	js.InternalObject(v.ptr).Call("$set", newSlice)
}

//gopherjs:replace
func (v Value) Slice(i, j int) Value {
	var (
		cap int
		typ *abi.Type
		s   *js.Object
	)
	switch kind := v.kind(); kind {
	case Array:
		if v.flag&flagAddr == 0 {
			panic("reflect.Value.Slice: slice of unaddressable array")
		}
		tt := v.typ().ArrayType()
		cap = int(tt.Len)
		typ = SliceOf(toRType(tt.Elem)).common()
		s = typ.JsType().New(v.object())

	case Slice:
		typ = v.typ()
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

	return makeValue(typ, js.Global.Call("$subslice", s, i, j), v.flag.ro())
}

//gopherjs:replace
func (v Value) Slice3(i, j, k int) Value {
	var (
		cap int
		typ *abi.Type
		s   *js.Object
	)
	switch kind := v.kind(); kind {
	case Array:
		if v.flag&flagAddr == 0 {
			panic("reflect.Value.Slice: slice of unaddressable array")
		}
		tt := v.typ().ArrayType()
		cap = int(tt.Len)
		typ = SliceOf(toRType(tt.Elem)).common()
		s = typ.JsType().New(v.object())

	case Slice:
		typ = v.typ()
		s = v.object()
		cap = s.Get("$capacity").Int()

	default:
		panic(&ValueError{"reflect.Value.Slice3", kind})
	}

	if i < 0 || j < i || k < j || k > cap {
		panic("reflect.Value.Slice3: slice index out of bounds")
	}

	return makeValue(typ, js.Global.Call("$subslice", s, i, j, k), v.flag.ro())
}

//gopherjs:replace
func (v Value) Close() {
	v.mustBe(Chan)
	v.mustBeExported()
	js.Global.Call("$close", v.object())
}

// typedslicecopy is implemented in prelude.js as $copySlice
//
//gopherjs:purge
func typedslicecopy(t *abi.Type, dst, src unsafeheader.Slice) int

// growslice is implemented in prelude.js as $growSlice.
//
//gopherjs:purge
func growslice(t *abi.Type, old unsafeheader.Slice, num int) unsafeheader.Slice

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

// gopherjs:replace
func noescape(p unsafe.Pointer) unsafe.Pointer {
	return p
}
