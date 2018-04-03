// +build js

package strconv

import (
	"github.com/gopherjs/gopherjs/js"
)

const maxInt32 float64 = 1<<31 - 1
const minInt32 float64 = -1 << 31

// Atoi returns the result of ParseInt(s, 10, 0) converted to type int.
func Atoi(s string) (int, error) {
	const fnAtoi = "Atoi"
	if len(s) == 0 {
		return 0, syntaxError(fnAtoi, s)
	}
	jsValue := js.Global.Call("Number", s, 10)
	if !js.Global.Call("isFinite", jsValue).Bool() {
		return 0, syntaxError(fnAtoi, s)
	}
	// Bounds checking
	floatval := jsValue.Float()
	if floatval > maxInt32 {
		return 1<<31 - 1, rangeError(fnAtoi, s)
	} else if floatval < minInt32 {
		return -1 << 31, rangeError(fnAtoi, s)
	}
	// Success!
	return jsValue.Int(), nil
}
