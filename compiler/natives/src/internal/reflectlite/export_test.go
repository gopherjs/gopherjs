//go:build js

package reflectlite

//gopherjs:replace
func Field(v Value, i int) Value { return v.Field(i) }

//gopherjs:purge Used in FirstMethodNameBytes
type EmbedWithUnexpMeth struct{}

//gopherjs:purge Used in FirstMethodNameBytes
type pinUnexpMeth interface{}

//gopherjs:purge Used in FirstMethodNameBytes
var pinUnexpMethI pinUnexpMeth

//gopherjs:purge Uses pointer arithmetic for names
func FirstMethodNameBytes(t Type) *byte
