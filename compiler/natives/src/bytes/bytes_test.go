//go:build js && linux

package bytes_test

import "testing"

//gopherjs:replace
func dangerousSlice(t *testing.T) []byte {
	t.Skip("dangerousSlice relies on syscall.Getpagesize, which GopherJS doesn't implement")

	panic("unreachable")
}
