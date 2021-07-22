//go:build js
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
