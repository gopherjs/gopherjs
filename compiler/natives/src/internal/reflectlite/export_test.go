//go:build js

package reflectlite

import (
	"unsafe"

	"internal/abi"

	"github.com/gopherjs/gopherjs/js"
)

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

//gopherjs:new
func (v Value) Field(i int) Value {
	tt := v.typ.StructType()
	if tt == nil {
		panic(&ValueError{"reflect.Value.Field", v.kind()})
	}

	if uint(i) >= uint(len(tt.Fields)) {
		panic("reflect: Field index out of range")
	}

	prop := jsType(v.typ).Get("fields").Index(i).Get("prop").String()
	field := &tt.Fields[i]
	typ := field.Typ

	fl := v.flag&(flagStickyRO|flagIndir|flagAddr) | flag(typ.Kind())
	if !field.Name.IsExported() {
		if field.Embedded() {
			fl |= flagEmbedRO
		} else {
			fl |= flagStickyRO
		}
	}

	if tag := tt.Fields[i].Name.Tag(); tag != "" && i != 0 {
		if jsTag := structTagGet(tag, `js`); jsTag != "" {
			for {
				v = v.Field(0)
				if v.typ == abi.JsObjectPtr {
					o := v.object().Get("object")
					return Value{typ, unsafe.Pointer(typ.JsPtrTo().New(
						js.InternalObject(func() *js.Object { return js.Global.Call("$internalize", o.Get(jsTag), jsType(typ)) }),
						js.InternalObject(func(x *js.Object) { o.Set(jsTag, js.Global.Call("$externalize", x, jsType(typ))) }),
					).Unsafe()), fl}
				}
				if v.typ.Kind() == abi.Pointer {
					v = v.Elem()
				}
			}
		}
	}

	s := js.InternalObject(v.ptr)
	if fl&flagIndir != 0 && typ.Kind() != abi.Array && typ.Kind() != abi.Struct {
		return Value{typ, unsafe.Pointer(typ.JsPtrTo().New(
			js.InternalObject(func() *js.Object { return abi.WrapJsObject(typ, s.Get(prop)) }),
			js.InternalObject(func(x *js.Object) { s.Set(prop, abi.UnwrapJsObject(typ, x)) }),
		).Unsafe()), fl}
	}
	return makeValue(typ, abi.WrapJsObject(typ, s.Get(prop)), fl)
}

// This is very similar to the `reflect.StructTag` methods `Get` and `Lookup`.
//
//gopherjs:new
func structTagGet(tag, key string) string {
	for tag != "" {
		// Skip leading space.
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// Scan to colon. A space, a quote or a control character is a syntax error.
		// Strictly speaking, control chars include the range [0x7f, 0x9f], not just
		// [0x00, 0x1f], but in practice, we ignore the multi-byte control characters
		// as it is simpler to inspect the tag's bytes than the tag's runes.
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		// Scan quoted string to find value.
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

		if key == name {
			value, syntaxErr := unquote(qvalue)
			if syntaxErr {
				break
			}
			return value
		}
	}
	return ""
}

//gopherjs:new Added to avoid a dependency on strconv.Unquote
func unquote(s string) (string, bool) {
	if len(s) < 2 {
		return s, false
	}
	if s[0] == '\'' || s[0] == '"' {
		if s[len(s)-1] == s[0] {
			return s[1 : len(s)-1], false
		}
		return "", true
	}
	return s, false
}

//gopherjs:purge Used in FirstMethodNameBytes
type EmbedWithUnexpMeth struct{}

//gopherjs:purge Used in FirstMethodNameBytes
type pinUnexpMeth interface{}

//gopherjs:purge Used in FirstMethodNameBytes
var pinUnexpMethI pinUnexpMeth

//gopherjs:purge Unused method that uses pointer arithmetic for names
func FirstMethodNameBytes(t Type) *byte
