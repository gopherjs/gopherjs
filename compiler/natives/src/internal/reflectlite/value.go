//go:build js

package reflectlite

import (
	"unsafe"

	"internal/abi"

	"github.com/gopherjs/gopherjs/js"
)

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
	panic(context + ": value of type " + toRType(v.typ).String() + " is not assignable to type " + toRType(dst).String())
}

var callHelper = js.Global.Get("$call").Interface().(func(...any) *js.Object)

func (v Value) call(op string, in []Value) []Value {
	var (
		t    *funcType
		fn   unsafe.Pointer
		rcvr *js.Object
	)
	if v.flag&flagMethod != 0 {
		_, t, fn = methodReceiver(op, v, int(v.flag)>>flagMethodShift)
		rcvr = v.object()
		if isWrapped(v.typ) {
			rcvr = jsType(v.typ).New(rcvr)
		}
	} else {
		t = (*funcType)(unsafe.Pointer(v.typ))
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
		if x.Kind() == abi.Invalid {
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
		targ := toRType(t.In(n))
		slice := MakeSlice(targ, m, m)
		elem := targ.Elem()
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
		targ := toRType(t.In(i))
		argsArray.SetIndex(i, unwrapJsObject(targ, arg.assignTo("reflect.Value.Call", targ.common(), nil).object()))
	}
	results := callHelper(js.InternalObject(fn), rcvr, argsArray)

	switch nout {
	case 0:
		return nil
	case 1:
		return []Value{makeValue(t.Out(0), wrapJsObject(toRType(t.Out(0)), results), 0)}
	default:
		ret := make([]Value, nout)
		for i := range ret {
			ret[i] = makeValue(t.Out(i), wrapJsObject(toRType(t.Out(i)), results.Index(i)), 0)
		}
		return ret
	}
}

func (v Value) Cap() int {
	k := v.kind()
	switch k {
	case abi.Array:
		return v.typ.Len()
	case abi.Chan, abi.Slice:
		return v.object().Get("$capacity").Int()
	}
	panic(&ValueError{Method: "reflect.Value.Cap", Kind: k})
}

func (v Value) Index(i int) Value {
	switch k := v.kind(); k {
	case abi.Array:
		tt := (*arrayType)(unsafe.Pointer(v.typ))
		if i < 0 || i > int(tt.Len) {
			panic("reflect: array index out of range")
		}
		typ := tt.Elem
		fl := v.flag&(flagIndir|flagAddr) | v.flag.ro() | flag(typ.Kind())

		a := js.InternalObject(v.ptr)
		rtyp := toRType(typ)
		if fl&flagIndir != 0 && typ.Kind() != abi.Array && typ.Kind() != abi.Struct {
			return Value{
				typ: typ,
				ptr: unsafe.Pointer(jsPtrTo(typ).New(
					js.InternalObject(func() *js.Object { return wrapJsObject(rtyp, a.Index(i)) }),
					js.InternalObject(func(x *js.Object) { a.SetIndex(i, unwrapJsObject(rtyp, x)) }),
				).Unsafe()),
				flag: fl,
			}
		}
		return makeValue(typ, wrapJsObject(rtyp, a.Index(i)), fl)

	case abi.Slice:
		s := v.object()
		if i < 0 || i >= s.Get("$length").Int() {
			panic("reflect: slice index out of range")
		}
		tt := (*sliceType)(unsafe.Pointer(v.typ))
		typ := tt.Elem
		fl := flagAddr | flagIndir | v.flag.ro() | flag(typ.Kind())

		i += s.Get("$offset").Int()
		a := s.Get("$array")
		rtyp := toRType(typ)
		if fl&flagIndir != 0 && typ.Kind() != abi.Array && typ.Kind() != abi.Struct {
			return Value{
				typ: typ,
				ptr: unsafe.Pointer(jsPtrTo(typ).New(
					js.InternalObject(func() *js.Object { return wrapJsObject(rtyp, a.Index(i)) }),
					js.InternalObject(func(x *js.Object) { a.SetIndex(i, unwrapJsObject(rtyp, x)) }),
				).Unsafe()),
				flag: fl,
			}
		}
		return makeValue(typ, wrapJsObject(rtyp, a.Index(i)), fl)

	case abi.String:
		str := *(*string)(v.ptr)
		if i < 0 || i >= len(str) {
			panic("reflect: string index out of range")
		}
		fl := v.flag.ro() | flag(abi.Uint8) | flagIndir
		c := str[i]
		return Value{
			typ:  uint8Type.Type,
			ptr:  unsafe.Pointer(&c),
			flag: fl,
		}

	default:
		panic(&ValueError{Method: "reflect.Value.Index", Kind: k})
	}
}

func (v Value) InterfaceData() [2]uintptr {
	panic("InterfaceData is not supported by GopherJS")
}

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
		panic(&ValueError{Method: "reflect.Value.IsNil", Kind: k})
	}
}

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
		panic(&ValueError{Method: "reflect.Value.Len", Kind: k})
	}
}

func (v Value) Pointer() uintptr {
	switch k := v.kind(); k {
	case abi.Chan, abi.Map, abi.Pointer, abi.UnsafePointer:
		if v.IsNil() {
			return 0
		}
		return v.object().Unsafe()
	case abi.Func:
		if v.IsNil() {
			return 0
		}
		return 1
	case abi.Slice:
		if v.IsNil() {
			return 0
		}
		return v.object().Get("$array").Unsafe()
	default:
		panic(&ValueError{Method: "reflect.Value.Pointer", Kind: k})
	}
}

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
			copyStruct(js.InternalObject(v.ptr), js.InternalObject(x.ptr), v.typ)
		default:
			js.InternalObject(v.ptr).Call("$set", x.object())
		}
		return
	}
	v.ptr = x.ptr
}

func (v Value) SetBytes(x []byte) {
	v.mustBeAssignable()
	v.mustBe(abi.Slice)
	rtyp := toRType(v.typ)
	if rtyp.Elem().Kind() != abi.Uint8 {
		panic("reflect.Value.SetBytes of non-byte slice")
	}
	slice := js.InternalObject(x)
	if rtyp.Name() != "" || rtyp.Elem().Name() != "" {
		typedSlice := jsType(v.typ).New(slice.Get("$array"))
		typedSlice.Set("$offset", slice.Get("$offset"))
		typedSlice.Set("$length", slice.Get("$length"))
		typedSlice.Set("$capacity", slice.Get("$capacity"))
		slice = typedSlice
	}
	js.InternalObject(v.ptr).Call("$set", slice)
}

func (v Value) SetCap(n int) {
	v.mustBeAssignable()
	v.mustBe(abi.Slice)
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
	v.mustBe(abi.Slice)
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
		s   *js.Object
	)
	switch kind := v.kind(); kind {
	case abi.Array:
		if v.flag&flagAddr == 0 {
			panic("reflect.Value.Slice: slice of unaddressable array")
		}
		tt := (*arrayType)(unsafe.Pointer(v.typ))
		cap = int(tt.Len)
		typ = SliceOf(toRType(tt.Elem))
		s = jsType(toAbiType(typ)).New(v.object())

	case abi.Slice:
		typ = toRType(v.typ)
		s = v.object()
		cap = s.Get("$capacity").Int()

	case abi.String:
		str := *(*string)(v.ptr)
		if i < 0 || j < i || j > len(str) {
			panic("reflect.Value.Slice: string slice index out of bounds")
		}
		return ValueOf(str[i:j])

	default:
		panic(&ValueError{Method: "reflect.Value.Slice", Kind: kind})
	}

	if i < 0 || j < i || j > cap {
		panic("reflect.Value.Slice: slice index out of bounds")
	}

	return makeValue(toAbiType(typ), js.Global.Call("$subslice", s, i, j), v.flag.ro())
}

func (v Value) Slice3(i, j, k int) Value {
	var (
		cap int
		typ Type
		s   *js.Object
	)
	switch kind := v.kind(); kind {
	case abi.Array:
		if v.flag&flagAddr == 0 {
			panic("reflect.Value.Slice: slice of unaddressable array")
		}
		tt := (*arrayType)(unsafe.Pointer(v.typ))
		cap = int(tt.Len)
		typ = SliceOf(toRType(tt.Elem))
		s = jsType(toAbiType(typ)).New(v.object())

	case abi.Slice:
		typ = toRType(v.typ)
		s = v.object()
		cap = s.Get("$capacity").Int()

	default:
		panic(&ValueError{Method: "reflect.Value.Slice3", Kind: kind})
	}

	if i < 0 || j < i || k < j || k > cap {
		panic("reflect.Value.Slice3: slice index out of bounds")
	}

	return makeValue(toAbiType(typ), js.Global.Call("$subslice", s, i, j, k), v.flag.ro())
}

func (v Value) Close() {
	v.mustBe(abi.Chan)
	v.mustBeExported()
	js.Global.Call("$close", v.object())
}

func (v Value) Elem() Value {
	switch k := v.kind(); k {
	case abi.Interface:
		val := v.object()
		if val == js.Global.Get("$ifaceNil") {
			return Value{}
		}
		typ := reflectType(val.Get("constructor"))
		return makeValue(toAbiType(typ), val.Get("$val"), v.flag.ro())

	case abi.Pointer:
		if v.IsNil() {
			return Value{}
		}
		val := v.object()
		tt := (*ptrType)(unsafe.Pointer(v.typ))
		fl := v.flag&flagRO | flagIndir | flagAddr
		fl |= flag(tt.Elem.Kind())
		return Value{
			typ:  tt.Elem,
			ptr:  unsafe.Pointer(wrapJsObject(toRType(tt.Elem), val).Unsafe()),
			flag: fl,
		}

	default:
		panic(&ValueError{Method: "reflect.Value.Elem", Kind: k})
	}
}

// NumField returns the number of fields in the struct v.
// It panics if v's Kind is not Struct.
func (v Value) NumField() int {
	v.mustBe(abi.Struct)
	tt := (*structType)(unsafe.Pointer(v.typ))
	return len(tt.Fields)
}

// MapIndex returns the value associated with key in the map v.
// It panics if v's Kind is not Map.
// It returns the zero Value if key is not found in the map or if v represents a nil map.
// As in Go, the key's value must be assignable to the map's key type.
func (v Value) MapIndex(key Value) Value {
	v.mustBe(abi.Map)
	tt := (*mapType)(unsafe.Pointer(v.typ))

	// Do not require key to be exported, so that DeepEqual
	// and other programs can use all the keys returned by
	// MapKeys as arguments to MapIndex. If either the map
	// or the key is unexported, though, the result will be
	// considered unexported. This is consistent with the
	// behavior for structs, which allow read but not write
	// of unexported fields.
	key = key.assignTo("reflect.Value.MapIndex", tt.Key, nil)

	var k unsafe.Pointer
	if key.flag&flagIndir != 0 {
		k = key.ptr
	} else {
		k = unsafe.Pointer(&key.ptr)
	}
	e := mapaccess(toRType(v.typ), v.pointer(), k)
	if e == nil {
		return Value{}
	}
	typ := tt.Elem
	fl := (v.flag | key.flag).ro()
	fl |= flag(typ.Kind())
	return copyVal(toRType(typ), fl, e)
}

func (v Value) Field(i int) Value {
	if v.kind() != Struct {
		panic(&ValueError{Method: "reflect.Value.Field", Kind: v.kind()})
	}
	tt := (*structType)(unsafe.Pointer(v.typ))
	if uint(i) >= uint(len(tt.Fields)) {
		panic("reflect: Field index out of range")
	}

	prop := jsType(v.typ).Get("fields").Index(i).Get("prop").String()
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
		if jsTag := getJsTag(tag); jsTag != "" {
			for {
				v = v.Field(0)
				if toRType(v.typ) == jsObjectPtr {
					o := v.object().Get("object")
					return Value{
						typ: typ,
						ptr: unsafe.Pointer(jsPtrTo(typ).New(
							js.InternalObject(func() *js.Object { return js.Global.Call("$internalize", o.Get(jsTag), jsType(typ)) }),
							js.InternalObject(func(x *js.Object) { o.Set(jsTag, js.Global.Call("$externalize", x, jsType(typ))) }),
						).Unsafe()),
						flag: fl,
					}
				}
				if v.typ.Kind() == abi.Pointer {
					v = v.Elem()
				}
			}
		}
	}

	s := js.InternalObject(v.ptr)
	if fl&flagIndir != 0 && typ.Kind() != abi.Array && typ.Kind() != abi.Struct {
		return Value{
			typ: typ,
			ptr: unsafe.Pointer(jsPtrTo(typ).New(
				js.InternalObject(func() *js.Object { return wrapJsObject(toRType(typ), s.Get(prop)) }),
				js.InternalObject(func(x *js.Object) { s.Set(prop, unwrapJsObject(toRType(typ), x)) }),
			).Unsafe()),
			flag: fl,
		}
	}
	return makeValue(typ, wrapJsObject(toRType(typ), s.Get(prop)), fl)
}
