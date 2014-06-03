// +build js

package net

import (
	"github.com/gopherjs/gopherjs/js"
)

func Listen(net, laddr string) (Listener, error) {
	js.Global.Call("$notSupported", "net")
	panic("unreachable")
}
