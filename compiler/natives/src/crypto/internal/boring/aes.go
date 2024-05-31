//go:build js
// +build js

package boring

import "crypto/internal/alias"

func anyOverlap(x, y []byte) bool {
	return alias.AnyOverlap()AnyOverlap(x, y)
}
