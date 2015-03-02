// +build js

package net

import (
	"runtime"
	"syscall"
)

func Listen(net, laddr string) (Listener, error) {
	panic(&runtime.NotSupportedError{"net"})
}

func (d *Dialer) Dial(network, address string) (Conn, error) {
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

func maxListenerBacklog() int {
	return syscall.SOMAXCONN
}
