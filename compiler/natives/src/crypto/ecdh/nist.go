//go:build js
// +build js

package ecdh

import (
	"crypto/internal/nistec"
	"io"
)

// temporarily replacement of `nistCurve[Point nistPoint[Point]]` for go1.20 without generics.
type nistCurve struct {
	name        string
	newPoint    func() nistec.WrappedPoint
	scalarOrder []byte
}

//gopherjs:override-signature
func (c *nistCurve) String() string

//gopherjs:override-signature
func (c *nistCurve) GenerateKey(rand io.Reader) (*PrivateKey, error)

//gopherjs:override-signature
func (c *nistCurve) NewPrivateKey(key []byte) (*PrivateKey, error)

//gopherjs:override-signature
func (c *nistCurve) privateKeyToPublicKey(key *PrivateKey) *PublicKey

//gopherjs:override-signature
func (c *nistCurve) NewPublicKey(key []byte) (*PublicKey, error)

//gopherjs:override-signature
func (c *nistCurve) ecdh(local *PrivateKey, remote *PublicKey) ([]byte, error)

//gopherjs:purge for go1.20 without generics
type nistPoint[T any] interface{}

// temporarily replacement for go1.20 without generics.
var p256 = &nistCurve{
	name:        "P-256",
	newPoint:    nistec.NewP256WrappedPoint,
	scalarOrder: p256Order,
}

// temporarily replacement for go1.20 without generics.
var p384 = &nistCurve{
	name:        "P-384",
	newPoint:    nistec.NewP384WrappedPoint,
	scalarOrder: p384Order,
}

// temporarily replacement for go1.20 without generics.
var p521 = &nistCurve{
	name:        "P-521",
	newPoint:    nistec.NewP521WrappedPoint,
	scalarOrder: p521Order,
}
