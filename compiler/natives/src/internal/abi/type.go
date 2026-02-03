package abi

import (
	"unsafe"

	"github.com/gopherjs/gopherjs/js"
)

//gopherjs:new
const (
	idJsType      = "_jsType"
	idReflectType = "_reflectType"
	idKindType    = "kindType"
	idRtype       = "_rtype"
)

//gopherjs:new
var UncommonTypeMap = make(map[*Type]*UncommonType)

//gopherjs:replace
func (t *Type) Uncommon() *UncommonType {
	return UncommonTypeMap[t]
}

//gopherjs:replace
type UncommonType struct {
	PkgPath NameOff // import path
	Mcount  uint16  // method count
	Xcount  uint16  // exported method count

	// GOPHERJS: Added access to methods
	Methods_ []Method
}

//gopherjs:replace
func (t *UncommonType) Methods() []Method {
	return t.Methods_
}

//gopherjs:replace
func (t *UncommonType) ExportedMethods() []Method {
	return t.Methods_[:t.Xcount:t.Xcount]
}

//gopherjs:replace
type FuncType struct {
	Type     `reflect:"func"`
	InCount  uint16
	OutCount uint16

	// GOPHERJS: Add references to in and out args
	In_  []*Type
	Out_ []*Type
}

//gopherjs:replace
func (t *FuncType) InSlice() []*Type {
	return t.In_
}

//gopherjs:replace
func (t *FuncType) OutSlice() []*Type {
	return t.Out_
}

//gopherjs:replace
type Name struct {
	name     string
	tag      string
	exported bool
	embedded bool
}

//gopherjs:replace
func (n Name) IsExported() bool { return n.exported }

//gopherjs:replace
func (n Name) HasTag() bool { return len(n.tag) > 0 }

//gopherjs:replace
func (n Name) IsEmbedded() bool { return n.embedded }

//gopherjs:replace
func (n Name) IsBlank() bool { return n.Name() == `_` }

//gopherjs:replace
func (n Name) Name() string { return n.name }

//gopherjs:replace
func (n Name) Tag() string { return n.tag }

//gopherjs:purge Used for byte encoding of name, not used in JS
func writeVarint(buf []byte, n int) int

//gopherjs:purge Used for byte encoding of name, not used in JS
func (n Name) DataChecked(off int, whySafe string) *byte

//gopherjs:purge Used for byte encoding of name, not used in JS
func (n Name) Data(off int) *byte

//gopherjs:purge Used for byte encoding of name, not used in JS
func (n Name) ReadVarint(off int) (int, int)

//gopherjs:replace
func NewName(n, tag string, exported, embedded bool) Name {
	return Name{
		name:     n,
		tag:      tag,
		exported: exported,
		embedded: embedded,
	}
}

//gopherjs:new
var nameOffList []Name

//gopherjs:new
func (typ *Type) NameOff(off NameOff) Name {
	return nameOffList[int(off)]
}

//gopherjs:new
func newNameOff(n Name) NameOff {
	i := len(nameOffList)
	nameOffList = append(nameOffList, n)
	return NameOff(i)
}

//gopherjs:new
var typeOffList []*Type

//gopherjs:new
func (typ *Type) TypeOff(off TypeOff) *Type {
	return typeOffList[int(off)]
}

//gopherjs:new
func newTypeOff(t *Type) TypeOff {
	i := len(typeOffList)
	typeOffList = append(typeOffList, t)
	return TypeOff(i)
}

//gopherjs:new
func (typ *Type) JsType() *js.Object {
	return js.InternalObject(typ).Get(idJsType)
}

//gopherjs:new
func (typ *Type) PtrTo() *Type {
	return ReflectType(js.Global.Call("$ptrType", typ.JsType()))
}

//gopherjs:new
func (typ *Type) JsPtrTo() *js.Object {
	return typ.PtrTo().JsType()
}

//gopherjs:new
func (typ *Type) IsWrapped() bool {
	return typ.JsType().Get("wrapped").Bool()
}

//gopherjs:new
func internalStr(strObj *js.Object) string {
	var c struct{ str string }
	js.InternalObject(c).Set("str", strObj) // get string without internalizing
	return c.str
}

//gopherjs:new
func ReflectType(typ *js.Object) *Type {
	if typ.Get(idReflectType) == js.Undefined {
		abiTyp := &Type{
			Size_: uintptr(typ.Get("size").Int()),
			Kind_: uint8(typ.Get("kind").Int()),
			Str:   newNameOff(NewName(internalStr(typ.Get("string")), "", typ.Get("exported").Bool(), false)),
		}
		js.InternalObject(abiTyp).Set(idJsType, typ)
		typ.Set(idReflectType, js.InternalObject(abiTyp))

		methodSet := js.Global.Call("$methodSet", typ)
		if methodSet.Length() != 0 || typ.Get("named").Bool() {
			abiTyp.TFlag |= TFlagUncommon
			if typ.Get("named").Bool() {
				abiTyp.TFlag |= TFlagNamed
			}
			var reflectMethods []Method
			for i := 0; i < methodSet.Length(); i++ { // Exported methods first.
				m := methodSet.Index(i)
				exported := internalStr(m.Get("pkg")) == ""
				if !exported {
					continue
				}
				reflectMethods = append(reflectMethods, Method{
					Name: newNameOff(NewName(internalStr(m.Get("name")), "", exported, false)),
					Mtyp: newTypeOff(ReflectType(m.Get("typ"))),
				})
			}
			xcount := uint16(len(reflectMethods))
			for i := 0; i < methodSet.Length(); i++ { // Unexported methods second.
				m := methodSet.Index(i)
				exported := internalStr(m.Get("pkg")) == ""
				if exported {
					continue
				}
				reflectMethods = append(reflectMethods, Method{
					Name: newNameOff(NewName(internalStr(m.Get("name")), "", exported, false)),
					Mtyp: newTypeOff(ReflectType(m.Get("typ"))),
				})
			}
			ut := &UncommonType{
				PkgPath:  newNameOff(NewName(internalStr(typ.Get("pkg")), "", false, false)),
				Mcount:   uint16(methodSet.Length()),
				Xcount:   xcount,
				Methods_: reflectMethods,
			}
			UncommonTypeMap[abiTyp] = ut
			js.InternalObject(ut).Set(idJsType, typ)
		}

		switch abiTyp.Kind() {
		case Array:
			setKindType(abiTyp, &ArrayType{
				Type: *abiTyp,
				Elem: ReflectType(typ.Get("elem")),
				Len:  uintptr(typ.Get("len").Int()),
			})
		case Chan:
			dir := BothDir
			if typ.Get("sendOnly").Bool() {
				dir = SendDir
			}
			if typ.Get("recvOnly").Bool() {
				dir = RecvDir
			}
			setKindType(abiTyp, &ChanType{
				Type: *abiTyp,
				Elem: ReflectType(typ.Get("elem")),
				Dir:  dir,
			})
		case Func:
			params := typ.Get("params")
			in := make([]*Type, params.Length())
			for i := range in {
				in[i] = ReflectType(params.Index(i))
			}
			results := typ.Get("results")
			out := make([]*Type, results.Length())
			for i := range out {
				out[i] = ReflectType(results.Index(i))
			}
			outCount := uint16(results.Length())
			if typ.Get("variadic").Bool() {
				outCount |= 1 << 15
			}
			setKindType(abiTyp, &FuncType{
				Type:     *abiTyp,
				InCount:  uint16(params.Length()),
				OutCount: outCount,
				In_:      in,
				Out_:     out,
			})
		case Interface:
			methods := typ.Get("methods")
			imethods := make([]Imethod, methods.Length())
			for i := range imethods {
				m := methods.Index(i)
				imethods[i] = Imethod{
					Name: newNameOff(NewName(internalStr(m.Get("name")), "", internalStr(m.Get("pkg")) == "", false)),
					Typ:  newTypeOff(ReflectType(m.Get("typ"))),
				}
			}
			setKindType(abiTyp, &InterfaceType{
				Type:    *abiTyp,
				PkgPath: NewName(internalStr(typ.Get("pkg")), "", false, false),
				Methods: imethods,
			})
		case Map:
			setKindType(abiTyp, &MapType{
				Type: *abiTyp,
				Key:  ReflectType(typ.Get("key")),
				Elem: ReflectType(typ.Get("elem")),
			})
		case Pointer:
			setKindType(abiTyp, &PtrType{
				Type: *abiTyp,
				Elem: ReflectType(typ.Get("elem")),
			})
		case Slice:
			setKindType(abiTyp, &SliceType{
				Type: *abiTyp,
				Elem: ReflectType(typ.Get("elem")),
			})
		case Struct:
			fields := typ.Get("fields")
			reflectFields := make([]StructField, fields.Length())
			for i := range reflectFields {
				f := fields.Index(i)
				reflectFields[i] = StructField{
					Name:   NewName(internalStr(f.Get("name")), internalStr(f.Get("tag")), f.Get("exported").Bool(), f.Get("embedded").Bool()),
					Typ:    ReflectType(f.Get("typ")),
					Offset: uintptr(i),
				}
			}
			setKindType(abiTyp, &StructType{
				Type:    *abiTyp,
				PkgPath: NewName(internalStr(typ.Get("pkgPath")), "", false, false),
				Fields:  reflectFields,
			})
		}
	}

	return (*Type)(unsafe.Pointer(typ.Get(idReflectType).Unsafe()))
}

//gopherjs:new
func setKindType(abiTyp *Type, kindType any) {
	js.InternalObject(abiTyp).Set(idKindType, js.InternalObject(kindType))
	js.InternalObject(kindType).Set(idRtype, js.InternalObject(abiTyp))
}
