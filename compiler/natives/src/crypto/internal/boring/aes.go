//go:build js
// +build js

package boring

import "github.com/gopherjs/gopherjs/compiler/natives/src/internal/alias"

func anyOverlap(x, y []byte) bool {
	return alias.AnyOverlap(x, y)
}
