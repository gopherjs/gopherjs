//go:build js

package reflectlite

import (
	"unsafe"

	"internal/abi"

	"github.com/gopherjs/gopherjs/js"
)

//gopherjs:replace
func methodName() string {
	// TODO(grantnelson-wf): methodName returns the name of the calling method,
	// assumed to be two stack frames above.
	return "?FIXME?"
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
		tt := v.typ.PtrType()
		fl := v.flag&flagRO | flagIndir | flagAddr
		fl |= flag(tt.Elem.Kind())
		return Value{
			typ:  tt.Elem,
			ptr:  unsafe.Pointer(abi.WrapJsObject(tt.Elem, val).Unsafe()),
			flag: fl,
		}

	default:
		panic(&ValueError{Method: "reflect.Value.Elem", Kind: k})
	}
}

//gopherjs:replace
func valueInterface(v Value) any {
	if v.flag == 0 {
		panic(&ValueError{Method: "reflect.Value.Interface", Kind: 0})
	}

	if v.flag&flagMethod != 0 {
		v = makeMethodValue("Interface", v)
	}

	if v.typ.IsWrapped() {
		if v.flag&flagIndir != 0 && v.Kind() == abi.Struct {
			cv := v.typ.JsType().Call("zero")
			abi.CopyStruct(cv, v.object(), v.typ)
			return any(unsafe.Pointer(v.typ.JsType().New(cv).Unsafe()))
		}
		return any(unsafe.Pointer(v.typ.JsType().New(v.object()).Unsafe()))
	}
	return any(unsafe.Pointer(v.object().Unsafe()))
}

//gopherjs:new This is new to reflectlite but there are commented out references in the native code and a copy in reflect.
func makeMethodValue(op string, v Value) Value {
	if v.flag&flagMethod == 0 {
		panic("reflect: internal error: invalid use of makePartialFunc")
	}

	fn := methodReceiver(op, v, int(v.flag)>>flagMethodShift)
	rcvr := v.object()
	if v.typ.IsWrapped() {
		rcvr = v.typ.JsType().New(rcvr)
	}
	fv := js.MakeFunc(func(this *js.Object, arguments []*js.Object) any {
		return js.InternalObject(fn).Call("apply", rcvr, arguments)
	})
	return Value{
		typ:  v.Type().common(),
		ptr:  unsafe.Pointer(fv.Unsafe()),
		flag: v.flag.ro() | flag(abi.Func),
	}
}

//gopherjs:new This is a simplified copy of the version in reflect.
func methodReceiver(op string, v Value, i int) (fn unsafe.Pointer) {
	var prop string
	if v.typ.Kind() == abi.Interface {
		tt := v.typ.InterfaceType()
		if i < 0 || i >= len(tt.Methods) {
			panic("reflect: internal error: invalid method index")
		}
		m := &tt.Methods[i]
		if !tt.NameOff(m.Name).IsExported() {
			panic("reflect: " + op + " of unexported method")
		}
		prop = tt.NameOff(m.Name).Name()
	} else {
		ms := v.typ.ExportedMethods()
		if uint(i) >= uint(len(ms)) {
			panic("reflect: internal error: invalid method index")
		}
		m := ms[i]
		if !v.typ.NameOff(m.Name).IsExported() {
			panic("reflect: " + op + " of unexported method")
		}
		prop = js.Global.Call("$methodSet", v.typ.JsType()).Index(i).Get("prop").String()
	}
	rcvr := v.object()
	if v.typ.IsWrapped() {
		rcvr = v.typ.JsType().New(rcvr)
	}
	fn = unsafe.Pointer(rcvr.Get(prop).Unsafe())
	return
}

//gopherjs:replace
func (v Value) IsNil() bool {
	switch k := v.kind(); k {
	case abi.Pointer, abi.Slice:
		return v.object() == v.typ.JsType().Get("nil")
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
		panic(&ValueError{Method: "reflect.Value.Len", Kind: k})
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
			v.typ.JsType().Call("copy", js.InternalObject(v.ptr), js.InternalObject(x.ptr))
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

//gopherjs:new This is added for export_test but is otherwise unused.
func (v Value) Field(i int) Value {
	tt := v.typ.StructType()
	if tt == nil {
		panic(&ValueError{Method: "reflect.Value.Field", Kind: v.kind()})
	}
	if uint(i) >= uint(len(tt.Fields)) {
		panic("reflect: Field index out of range")
	}

	prop := v.typ.JsType().Get("fields").Index(i).Get("prop").String()
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
				if abi.IsJsObjectPtr(v.typ) {
					o := v.object().Get("object")
					return Value{
						typ: typ,
						ptr: unsafe.Pointer(typ.JsPtrTo().New(
							js.InternalObject(func() *js.Object { return js.Global.Call("$internalize", o.Get(jsTag), typ.JsType()) }),
							js.InternalObject(func(x *js.Object) { o.Set(jsTag, js.Global.Call("$externalize", x, typ.JsType())) }),
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
			ptr: unsafe.Pointer(typ.JsPtrTo().New(
				js.InternalObject(func() *js.Object { return abi.WrapJsObject(typ, s.Get(prop)) }),
				js.InternalObject(func(x *js.Object) { s.Set(prop, abi.UnwrapJsObject(typ, x)) }),
			).Unsafe()),
			flag: fl,
		}
	}
	return makeValue(typ, abi.WrapJsObject(typ, s.Get(prop)), fl)
}

//gopherjs:replace
func unsafe_New(typ *abi.Type) unsafe.Pointer {
	return abi.UnsafeNew(typ)
}

//gopherjs:replace
func ValueOf(i any) Value {
	if i == nil {
		return Value{}
	}
	return makeValue(abi.ReflectType(js.InternalObject(i).Get("constructor")), js.InternalObject(i).Get("$val"), 0)
}

//gopherjs:new
func makeValue(t *abi.Type, v *js.Object, fl flag) Value {
	switch t.Kind() {
	case abi.Array, abi.Struct, abi.Pointer:
		return Value{
			typ:  t,
			ptr:  unsafe.Pointer(v.Unsafe()),
			flag: fl | flag(t.Kind()),
		}
	}
	return Value{
		typ:  t,
		ptr:  unsafe.Pointer(js.Global.Call("$newDataPointer", v, t.JsPtrTo()).Unsafe()),
		flag: fl | flag(t.Kind()) | flagIndir,
	}
}

//gopherjs:new
func (v Value) object() *js.Object {
	if v.typ.Kind() == abi.Array || v.typ.Kind() == abi.Struct {
		return js.InternalObject(v.ptr)
	}
	if v.flag&flagIndir != 0 {
		val := js.InternalObject(v.ptr).Call("$get")
		jsTyp := v.typ.JsType()
		if val != js.Global.Get("$ifaceNil") && val.Get("constructor") != jsTyp {
			switch v.typ.Kind() {
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
	case directlyAssignable(dst, v.typ):
		// Overwrite type so that they match.
		// Same memory layout, so no harm done.
		fl := v.flag&(flagAddr|flagIndir) | v.flag.ro()
		fl |= flag(dst.Kind())
		return Value{
			typ:  dst,
			ptr:  v.ptr,
			flag: fl,
		}

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
		return Value{
			typ:  dst,
			ptr:  target,
			flag: flagIndir | flag(Interface),
		}
	}

	// Failed.
	panic(context + ": value of type " + toRType(v.typ).String() + " is not assignable to type " + toRType(dst).String())
}

//gopherjs:replace
func ifaceE2I(t *abi.Type, src any, dst unsafe.Pointer) {
	abi.IfaceE2I(t, src, dst)
}

//gopherjs:replace
func typedmemmove(t *abi.Type, dst, src unsafe.Pointer) {
	abi.TypedMemMove(t, dst, src)
}
