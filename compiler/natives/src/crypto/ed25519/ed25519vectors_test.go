//go:build js
// +build js

package ed25519_test

import "testing"

func TestEd25519Vectors(t *testing.T) {
	t.Skip("exec.Command() is not supported by GopherJS")
}
