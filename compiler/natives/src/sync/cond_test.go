//go:build js
// +build js

package sync_test

import "testing"

func TestCondCopy(t *testing.T) {
	t.Skip("Copy checker requires raw pointers, which GopherJS doesn't fully support.")
}
