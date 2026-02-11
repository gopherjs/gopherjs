//go:build js

package abi_test

import "testing"

//gopherjs:replace
func TestFuncPC(t *testing.T) {
	t.Skip(`test involes checking the PC (program counter)`)
}

//gopherjs:replace
func TestFuncPCCompileError(t *testing.T) {
	t.Skip(`test involes checking the PC (program counter)`)
}
