//go:build js

package reflectlite

import (
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
func toAbiType(t Type) *abi.Type {
	return t.(rtype).common()
}

//gopherjs:new
func jsType(t Type) *js.Object {
	return toAbiType(t).JsType()
}

//gopherjs:new
func jsPtrTo(t Type) *js.Object {
	return toAbiType(t).JsPtrTo()
}
