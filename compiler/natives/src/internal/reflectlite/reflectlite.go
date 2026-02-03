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
func toAbiType(typ Type) *abi.Type {
	return typ.(rtype).common()
}

//gopherjs:new
func jsType(t Type) *js.Object {
	return toAbiType(t).JsType()
}

//gopherjs:new
func jsPtrTo(t Type) *js.Object {
	return toAbiType(t).JsPtrTo()
}

//gopherjs:new
var jsObjectPtr = abi.ReflectType(js.Global.Get("$jsObjectPtr"))

//gopherjs:new
func wrapJsObject(typ *abi.Type, val *js.Object) *js.Object {
	if typ == jsObjectPtr {
		return jsObjectPtr.JsType().New(val)
	}
	return val
}

//gopherjs:new
func unwrapJsObject(typ *abi.Type, val *js.Object) *js.Object {
	if typ == jsObjectPtr {
		return val.Get("object")
	}
	return val
}

//gopherjs:new
func getJsTag(tag string) string {
	for tag != "" {
		// skip leading space
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// scan to colon.
		// a space or a quote is a syntax error
		i = 0
		for i < len(tag) && tag[i] != ' ' && tag[i] != ':' && tag[i] != '"' {
			i++
		}
		if i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		// scan quoted string to find value
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		if name == "js" {
			value, _ := unquote(qvalue)
			return value
		}
	}
	return ""
}
