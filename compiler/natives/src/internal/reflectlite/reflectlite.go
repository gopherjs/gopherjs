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
	used(funcType{})
	used(interfaceType{})
	used(structType{})
}

// gopherjs:new
func jsType(typ *abi.Type) *js.Object {
	return typ.JsType()
}

//gopherjs:purge Unused, replaced by abi.Name.
type name struct {
	bytes *byte
}

//gopherjs:replace
func pkgPath(n abi.Name) string {
	return n.PkgPath()
}

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
		return Value{t, unsafe.Pointer(v.Unsafe()), fl | flag(t.Kind())}
	}
	return Value{t, unsafe.Pointer(js.Global.Call("$newDataPointer", v, t.JsPtrTo()).Unsafe()), fl | flag(t.Kind()) | flagIndir}
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
		prop = js.Global.Call("$methodSet", jsType(v.typ)).Index(i).Get("prop").String()
	}
	rcvr := v.object()
	if v.typ.IsWrapped() {
		rcvr = jsType(v.typ).New(rcvr)
	}
	fn = unsafe.Pointer(rcvr.Get(prop).Unsafe())
	return
}

//gopherjs:replace
func valueInterface(v Value) any {
	if v.flag == 0 {
		panic(&ValueError{"reflect.Value.Interface", 0})
	}

	if v.flag&flagMethod != 0 {
		v = makeMethodValue("Interface", v)
	}

	if v.typ.IsWrapped() {
		if v.flag&flagIndir != 0 && v.Kind() == abi.Struct {
			cv := jsType(v.typ).Call("zero")
			abi.CopyStruct(cv, v.object(), v.typ)
			return any(unsafe.Pointer(jsType(v.typ).New(cv).Unsafe()))
		}
		return any(unsafe.Pointer(jsType(v.typ).New(v.object()).Unsafe()))
	}
	return any(unsafe.Pointer(v.object().Unsafe()))
}

//gopherjs:replace
func ifaceE2I(t *abi.Type, src any, dst unsafe.Pointer) {
	abi.IfaceE2I(t, src, dst)
}

// TODO(grantnelson-wf): methodName returns the name of the calling method,
// assumed to be two stack frames above. Determine if we can get this value now
// and if methodName is needed
//
//gopherjs:replace
func methodName() string {
	return "?FIXME?"
}

//gopherjs:new
func makeMethodValue(op string, v Value) Value {
	if v.flag&flagMethod == 0 {
		panic("reflect: internal error: invalid use of makePartialFunc")
	}

	fn := methodReceiver(op, v, int(v.flag)>>flagMethodShift)
	rcvr := v.object()
	if v.typ.IsWrapped() {
		rcvr = jsType(v.typ).New(rcvr)
	}
	fv := js.MakeFunc(func(this *js.Object, arguments []*js.Object) any {
		return js.InternalObject(fn).Call("apply", rcvr, arguments)
	})
	return Value{v.Type().common(), unsafe.Pointer(fv.Unsafe()), v.flag.ro() | flag(abi.Func)}
}
