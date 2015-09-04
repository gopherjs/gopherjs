package jsbuiltin

import (
	"github.com/gopherjs/gopherjs/js"
)

// Wrappers for standard JS Builtin functions

func DecodeURI(uri string) string {
	return js.Global.Call("decodeURI", uri).String()
}

func EncodeURI(uri string) string {
	return js.Global.Call("encodeURI", uri).String()
}

func EncodeURIComponent(uri string) string {
	return js.Global.Call("encodeURIComponent", uri).String()
}

func DecodeURIComponent(uri string) string {
	return js.Global.Call("decodeURIComponent", uri).String()
}

func IsFinite(value interface{}) bool {
	return js.Global.Call("isFinite", value).Bool()
}

func IsNaN(value interface{}) bool {
	return js.Global.Call("isNaN", value).Bool()
}
