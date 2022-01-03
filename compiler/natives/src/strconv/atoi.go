//go:build js
// +build js

package strconv

import (
	"github.com/gopherjs/gopherjs/js"
)

const (
	maxInt32 float64 = 1<<31 - 1
	minInt32 float64 = -1 << 31
)

// Atoi returns the result of ParseInt(s, 10, 0) converted to type int.
func Atoi(s string) (int, error) {
	const fnAtoi = "Atoi"
	if len(s) == 0 {
		return 0, syntaxError(fnAtoi, s)
	}
	// Investigate the bytes of the string
	// Validate each byte is allowed in parsing
	// Number allows some prefixes that Go does not: "0x" "0b", "0o"
	// additionally Number accepts decimals where Go does not "10.2"
	for i := 0; i < len(s); i++ {
		v := s[i]

		if v < '0' || v > '9' {
			if v != '+' && v != '-' {
				return 0, syntaxError(fnAtoi, s)
			}
		}
	}
	jsValue := js.Global.Call("Number", s, 10)
	if !js.Global.Call("isFinite", jsValue).Bool() {
		return 0, syntaxError(fnAtoi, s)
	}
	// Bounds checking
	floatval := jsValue.Float()
	if floatval > maxInt32 {
		return int(maxInt32), rangeError(fnAtoi, s)
	} else if floatval < minInt32 {
		return int(minInt32), rangeError(fnAtoi, s)
	}
	// Success!
	return jsValue.Int(), nil
}
