//go:build js
// +build js

package aes

import "github.com/gopherjs/gopherjs/compiler/natives/src/internal/alias"

func anyOverlap(x, y []byte) bool {
	return alias.AnyOverlap(x, y)
}

func inexactOverlap(x, y []byte) bool {
	return alias.InexactOverlap(x, y)
}
