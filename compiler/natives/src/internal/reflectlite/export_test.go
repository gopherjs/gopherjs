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

//gopherjs:purge Used in FirstMethodNameBytes
type EmbedWithUnexpMeth struct{}

//gopherjs:purge Used in FirstMethodNameBytes
type pinUnexpMeth interface{}

//gopherjs:purge Used in FirstMethodNameBytes
var pinUnexpMethI pinUnexpMeth

//gopherjs:purge Uses pointer arithmetic for names
func FirstMethodNameBytes(t Type) *byte
