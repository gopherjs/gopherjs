//go:build js
// +build js

package aes

//go:linkname anyOverlap internal/alias.AnyOverlap
func anyOverlap(x, y []byte) bool

//go:linkname inexactOverlap internal/alias.InexactOverlap
func inexactOverlap(x, y []byte) bool
