//go:build js
// +build js

package elliptic

import (
	"crypto/internal/nistec"
	"math/big"
)

// nistPoint uses generics so must be removed for generic-less GopherJS.
// All the following code changes in this file are to make p224, p256,
// p521, and p384 still function correctly without this generic struct.
//
//gopherjs:purge for go1.19 without generics
type nistPoint[T any] interface{}

// nistCurve replaces the generics with a version using the wrappedPoint
// interface, then update all the method signatures to also use wrappedPoint.
type nistCurve struct {
	newPoint func() nistec.WrappedPoint
	params   *CurveParams
}

//gopherjs:override-signature
func (curve *nistCurve) Params() *CurveParams

//gopherjs:override-signature
func (curve *nistCurve) IsOnCurve(x, y *big.Int) bool

//gopherjs:override-signature
func (curve *nistCurve) pointFromAffine(x, y *big.Int) (p nistec.WrappedPoint, err error)

//gopherjs:override-signature
func (curve *nistCurve) pointToAffine(p nistec.WrappedPoint) (x, y *big.Int)

//gopherjs:override-signature
func (curve *nistCurve) Add(x1, y1, x2, y2 *big.Int) (*big.Int, *big.Int)

//gopherjs:override-signature
func (curve *nistCurve) Double(x1, y1 *big.Int) (*big.Int, *big.Int)

//gopherjs:override-signature
func (curve *nistCurve) normalizeScalar(scalar []byte) []byte

//gopherjs:override-signature
func (curve *nistCurve) ScalarMult(Bx, By *big.Int, scalar []byte) (*big.Int, *big.Int)

//gopherjs:override-signature
func (curve *nistCurve) ScalarBaseMult(scalar []byte) (*big.Int, *big.Int)

//gopherjs:override-signature
func (curve *nistCurve) CombinedMult(Px, Py *big.Int, s1, s2 []byte) (x, y *big.Int)

//gopherjs:override-signature
func (curve *nistCurve) Unmarshal(data []byte) (x, y *big.Int)

//gopherjs:override-signature
func (curve *nistCurve) UnmarshalCompressed(data []byte) (x, y *big.Int)

var p224 = &nistCurve{
	newPoint: nistec.NewP224WrappedPoint,
}

type p256Curve struct {
	nistCurve
}

var p256 = &p256Curve{
	nistCurve: nistCurve{
		newPoint: nistec.NewP256WrappedPoint,
	},
}

var p521 = &nistCurve{
	newPoint: nistec.NewP521WrappedPoint,
}

var p384 = &nistCurve{
	newPoint: nistec.NewP384WrappedPoint,
}
