//go:build js

package io_test

import (
	"testing"
)

//gopherjs:replace
func TestMultiWriter_WriteStringSingleAlloc(t *testing.T) {
	t.Skip("testing.AllocsPerRun not supported in GopherJS")
}

//gopherjs:replace
func TestMultiReaderFreesExhaustedReaders(t *testing.T) {
	t.Skip("test relies on runtime.SetFinalizer, which GopherJS does not implement")
}

//gopherjs:replace
func TestCopyLargeWriter(t *testing.T) {
	// This test actually behaves more or less correctly, but it triggers a
	// different code path that panics instead of returning an error due to a bug
	// referenced below.
	t.Skip("https://github.com/gopherjs/gopherjs/issues/1003")
}
