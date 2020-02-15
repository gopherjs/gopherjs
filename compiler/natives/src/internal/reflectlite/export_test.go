// +build js

package reflectlite

import (
	"unsafe"
)

// Field returns the i'th field of the struct v.
// It panics if v's Kind is not Struct or i is out of range.
func Field(v Value, i int) Value {
	if v.kind() != Struct {
		panic(&ValueError{"reflect.Value.Field", v.kind()})
	}
	return v.Field(i)
}

func TField(typ Type, i int) Type {
	t := typ.(*rtype)
	if t.Kind() != Struct {
		panic("reflect: Field of non-struct type")
	}
	tt := (*structType)(unsafe.Pointer(t))
	return StructFieldType(tt, i)
}

// Field returns the i'th struct field.
func StructFieldType(t *structType, i int) Type {
	if i < 0 || i >= len(t.fields) {
		panic("reflect: Field index out of bounds")
	}
	p := &t.fields[i]
	return toType(p.typ)
}

// // Zero returns a Value representing the zero value for the specified type.
// // The result is different from the zero value of the Value struct,
// // which represents no value at all.
// // For example, Zero(TypeOf(42)) returns a Value with Kind Int and value 0.
// // The returned value is neither addressable nor settable.
// func Zero(typ Type) Value {
// 	if typ == nil {
// 		panic("reflect: Zero(nil)")
// 	}
// 	t := typ.(*rtype)
// 	fl := flag(t.Kind())
// 	if ifaceIndir(t) {
// 		return Value{t, unsafe_New(t), fl | flagIndir}
// 	}
// 	return Value{t, nil, fl}
// }

// // ToInterface returns v's current value as an interface{}.
// // It is equivalent to:
// //	var i interface{} = (v's underlying value)
// // It panics if the Value was obtained by accessing
// // unexported struct fields.
// func ToInterface(v Value) (i interface{}) {
// 	return valueInterface(v)
// }

// type EmbedWithUnexpMeth struct{}

// func (EmbedWithUnexpMeth) f() {}

// type pinUnexpMeth interface {
// 	f()
// }

// var pinUnexpMethI = pinUnexpMeth(EmbedWithUnexpMeth{})

// func FirstMethodNameBytes(t Type) *byte {
// 	_ = pinUnexpMethI

// 	ut := t.uncommon()
// 	if ut == nil {
// 		panic("type has no methods")
// 	}
// 	m := ut.methods()[0]
// 	mname := t.(*rtype).nameOff(m.name)
// 	if *mname.data(0, "name flag field")&(1<<2) == 0 {
// 		panic("method name does not have pkgPath *string")
// 	}
// 	return mname.bytes
// }

// type Buffer struct {
// 	buf []byte
// }
