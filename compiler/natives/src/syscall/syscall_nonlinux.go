//go:build js && !linux
// +build js,!linux

package syscall

const exitTrap = SYS_EXIT
