//go:build js

package reflectlite

import (
	"unsafe"

	"internal/abi"

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
}

//gopherjs:new
func jsType(t Type) *js.Object {
	return toAbiType(t).JsType()
}

//gopherjs:new
func toAbiType(t Type) *abi.Type {
	return t.(rtype).common()
}

//gopherjs:new
func jsPtrTo(t Type) *js.Object {
	return toAbiType(t).JsPtrTo()
}

//gopherjs:purge The name type is mostly unused, replaced by abi.Name, except in pkgPath which we don't implement.
type name struct{}

//gopherjs:replace
func pkgPath(n abi.Name) string { return "" }

//gopherjs:purge Unused function because of nameOffList in internal/abi overrides
func resolveNameOff(ptrInModule unsafe.Pointer, off int32) unsafe.Pointer

//gopherjs:purge Unused function because of typeOffList in internal/abi overrides
func resolveTypeOff(rtype unsafe.Pointer, off int32) unsafe.Pointer

//gopherjs:replace
func (t rtype) nameOff(off nameOff) abi.Name {
	return t.NameOff(off)
}

//gopherjs:replace
func (t rtype) typeOff(off typeOff) *abi.Type {
	return t.TypeOff(off)
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

//gopherjs:replace
func TypeOf(i any) Type {
	if i == nil {
		return nil
	}
	return toRType(abi.ReflectType(js.InternalObject(i).Get("constructor")))
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

//gopherjs:replace
func typedmemmove(t *abi.Type, dst, src unsafe.Pointer) {
	abi.TypedMemMove(t, dst, src)
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

//gopherjs:replace
func ifaceE2I(t *abi.Type, src any, dst unsafe.Pointer) {
	abi.IfaceE2I(t, src, dst)
}

//gopherjs:replace
func methodName() string {
	// TODO(grantnelson-wf): methodName returns the name of the calling method,
	// assumed to be two stack frames above.
	return "?FIXME?"
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
