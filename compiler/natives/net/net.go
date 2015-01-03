// +build js

package net

import (
	"runtime"
)

func Listen(net, laddr string) (Listener, error) {
	panic(&runtime.NotSupportedError{"net"})
}

func sysInit() {
}

func probeIPv4Stack() bool {
	return false
}

func probeIPv6Stack() (supportsIPv6, supportsIPv4map bool) {
	return false, false
}
