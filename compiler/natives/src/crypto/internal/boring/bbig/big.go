//go:build js
// +build js

package bbig

import (
	"crypto/internal/boring"
	"math/big"
)

func Enc(b *big.Int) boring.BigInt {
	if b == nil {
		return nil
	}
	x := b.Bits()
	if len(x) == 0 {
		return boring.BigInt{}
	}
	// Replacing original which uses unsafe:
	// return unsafe.Slice((*uint)(&x[0]), len(x))
	b2 := make(boring.BigInt, len(x))
	for i, w := range x {
		b2[i] = uint(w)
	}
	return b2
}

func Dec(b boring.BigInt) *big.Int {
	if b == nil {
		return nil
	}
	if len(b) == 0 {
		return new(big.Int)
	}
	// Replacing original which uses unsafe:
	//x := unsafe.Slice((*big.Word)(&b[0]), len(b))
	x := make([]big.Word, len(b))
	for i, w := range b {
		x[i] = big.Word(w)
	}
	return new(big.Int).SetBits(x)
}
