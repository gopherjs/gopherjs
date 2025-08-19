//go:build legacy_syscall && gopherjs

// This program tests GopherJS's ability to perform raw syscalls using the
// deprecated node_syscall extension. See TestLegacySyscall.
package main

import (
	"syscall"
	"unsafe"
)

func main() {
	msg := []byte("Hello, world!\n")
	_, _, errno := syscall.Syscall(1 /* SYS_WRITE on Linux */, 1 /* stdout */, uintptr(unsafe.Pointer(&msg[0])), uintptr(len(msg)))
	if errno != 0 {
		println(errno.Error())
	}
}
