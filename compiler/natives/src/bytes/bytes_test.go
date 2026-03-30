//go:build js

package bytes_test

import "testing"

func dangerousSlice(t *testing.T) []byte {
	t.Skip("dangerousSlice relies on syscall.Getpagesize, which GopherJS doesn't implement")

	panic("unreachable")
}

//gopherjs:replace
func TestIssue65571(t *testing.T) {
	t.Skip("TestIssue65571 expects int to be greater than 32 bits")
}
