//go:build js

package abi

import (
	"unsafe"

	"github.com/gopherjs/gopherjs/js"
)

// GOPHERJS: These utils are being added because they are common between
// reflect and reflectlite. The [Go proverb](https://go-proverbs.github.io/),
// "A little copying is better than a little dependency," isn't applicable
// when both reflect and reflectlite already depend on ABI. We can reduce
// our native overrides in both locations by putting common code here.

//gopherjs:new
type errorString struct {
	s string
}

//gopherjs:new
func (e *errorString) Error() string {
	return e.s
}

//gopherjs:new
var ErrSyntax = &errorString{"invalid syntax"}

//gopherjs:new Added to avoid a dependency on strconv.Unquote
func unquote(s string) (string, error) {
	if len(s) < 2 {
		return s, nil
	}
	if s[0] == '\'' || s[0] == '"' {
		if s[len(s)-1] == s[0] {
			return s[1 : len(s)-1], nil
		}
		return "", ErrSyntax
	}
	return s, nil
}

//gopherjs:new
func GetJsTag(tag string) string {
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

// NewMethodName creates name instance for a method.
//
// Input object is expected to be an entry of the "methods" list of the
// corresponding JS type.
//
//gopherjs:new
func NewMethodName(m *js.Object) Name {
	return Name{
		name:     internalStr(m.Get("name")),
		tag:      "",
		pkgPath:  internalStr(m.Get("pkg")),
		exported: internalStr(m.Get("pkg")) == "",
	}
}

//gopherjs:new
func UnsafeNew(typ *Type) unsafe.Pointer {
	switch typ.Kind() {
	case Struct:
		return unsafe.Pointer(typ.JsType().Get("ptr").New().Unsafe())
	case Array:
		return unsafe.Pointer(typ.JsType().Call("zero").Unsafe())
	default:
		return unsafe.Pointer(js.Global.Call("$newDataPointer", typ.JsType().Call("zero"), typ.JsPtrTo()).Unsafe())
	}
}

//gopherjs:new
func IfaceE2I(t *Type, src any, dst unsafe.Pointer) {
	js.InternalObject(dst).Call("$set", js.InternalObject(src))
}

//gopherjs:new
func TypedMemMove(t *Type, dst, src unsafe.Pointer) {
	js.InternalObject(dst).Call("$set", js.InternalObject(src).Call("$get"))
}
