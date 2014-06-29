// +build js

package net

import (
	"runtime"
)

func Listen(net, laddr string) (Listener, error) {
	panic(&runtime.NotSupportedError{"net"})
}
