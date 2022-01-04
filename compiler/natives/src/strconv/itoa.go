//go:build js
// +build js

package strconv

import (
	"github.com/gopherjs/gopherjs/js"
)

// Itoa in gopherjs is always a 32bit int so the native toString
// always handles it successfully.
func Itoa(i int) string {
	return js.InternalObject(i).Call("toString").String()
}
