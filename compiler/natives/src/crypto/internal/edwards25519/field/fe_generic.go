package field

import (
	_ "unsafe" // For go:linkname
)

// feMulGeneric computes `v = a * b mod 2^255-19`
//
//go:linkname feMul crypto/internal/edwards25519/field.feMulGeneric_js
//gopherjs:force-non-blocking
func feMul(v, a, b *Element)
