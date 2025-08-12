//go:build js

package sync_test

import "testing"

//gopherjs:replace
func TestCondCopy(t *testing.T) {
	t.Skip("Copy checker requires raw pointers, which GopherJS doesn't fully support.")
}
