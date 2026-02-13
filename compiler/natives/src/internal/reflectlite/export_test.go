//go:build js

package reflectlite

// Field returns the i'th field of the struct v.
// It panics if v's Kind is not Struct or i is out of range.
//
//gopherjs:replace
func Field(v Value, i int) Value {
	if v.kind() != Struct {
		panic(&ValueError{"reflect.Value.Field", v.kind()})
	}
	return v.Field(i)
}

//gopherjs:replace Used a pointer cast for the struct kind type.
func TField(typ Type, i int) Type {
	tt := toAbiType(typ).StructType()
	if tt == nil {
		panic("reflect: Field of non-struct type")
	}
	return StructFieldType(tt, i)
}

//gopherjs:purge Used in FirstMethodNameBytes
type EmbedWithUnexpMeth struct{}

//gopherjs:purge Used in FirstMethodNameBytes
type pinUnexpMeth interface{}

//gopherjs:purge Used in FirstMethodNameBytes
var pinUnexpMethI pinUnexpMeth

//gopherjs:purge Uses pointer arithmetic for names
func FirstMethodNameBytes(t Type) *byte
