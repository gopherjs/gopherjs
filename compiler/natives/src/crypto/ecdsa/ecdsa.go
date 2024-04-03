//go:build js
// +build js

package ecdsa

import (
	"crypto/elliptic"
	"crypto/internal/bigmod"
	"crypto/internal/nistec"
	"io"
	"math/big"
)

//gopherjs:override-signature
func generateNISTEC(c *nistCurve, rand io.Reader) (*PrivateKey, error)

//gopherjs:override-signature
func randomPoint(c *nistCurve, rand io.Reader) (k *bigmod.Nat, p nistec.WrappedPoint, err error)

//gopherjs:override-signature
func signNISTEC(c *nistCurve, priv *PrivateKey, csprng io.Reader, hash []byte) (sig []byte, err error)

//gopherjs:override-signature
func inverse(c *nistCurve, kInv, k *bigmod.Nat)

//gopherjs:override-signature
func hashToNat(c *nistCurve, e *bigmod.Nat, hash []byte)

//gopherjs:override-signature
func verifyNISTEC(c *nistCurve, pub *PublicKey, hash, sig []byte) bool

//gopherjs:purge for go1.20 without generics
type nistPoint[T any] interface{}

// temporarily replacement of `nistCurve[Point nistPoint[Point]]` for go1.20 without generics.
type nistCurve struct {
	newPoint func() nistec.WrappedPoint
	curve    elliptic.Curve
	N        *bigmod.Modulus
	nMinus2  []byte
}

//gopherjs:override-signature
func (curve *nistCurve) pointFromAffine(x, y *big.Int) (p nistec.WrappedPoint, err error)

//gopherjs:override-signature
func (curve *nistCurve) pointToAffine(p nistec.WrappedPoint) (x, y *big.Int, err error)

var _p224 *nistCurve

func p224() *nistCurve {
	p224Once.Do(func() {
		_p224 = &nistCurve{
			newPoint: nistec.NewP224WrappedPoint,
		}
		precomputeParams(_p224, elliptic.P224())
	})
	return _p224
}

var _p256 *nistCurve

func p256() *nistCurve {
	p256Once.Do(func() {
		_p256 = &nistCurve{
			newPoint: nistec.NewP256WrappedPoint,
		}
		precomputeParams(_p256, elliptic.P256())
	})
	return _p256
}

var _p384 *nistCurve

func p384() *nistCurve {
	p384Once.Do(func() {
		_p384 = &nistCurve{
			newPoint: nistec.NewP384WrappedPoint,
		}
		precomputeParams(_p384, elliptic.P384())
	})
	return _p384
}

var _p521 *nistCurve

func p521() *nistCurve {
	p521Once.Do(func() {
		_p521 = &nistCurve{
			newPoint: nistec.NewP521WrappedPoint,
		}
		precomputeParams(_p521, elliptic.P521())
	})
	return _p521
}

//gopherjs:override-signature
func precomputeParams(c *nistCurve, curve elliptic.Curve)
