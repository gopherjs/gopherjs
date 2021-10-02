//go:build js && !linux && !wasm
// +build js,!linux,!wasm

package syscall

const exitTrap = SYS_EXIT
