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

type wrappedPoint interface {
	Bytes() []byte
	SetBytes(b []byte) (wrappedPoint, error)
	Add(w1, w2 wrappedPoint) wrappedPoint
	Double(w1 wrappedPoint) wrappedPoint
	ScalarMult(w1 wrappedPoint, scalar []byte) (wrappedPoint, error)
	ScalarBaseMult(scalar []byte) (wrappedPoint, error)
}

// nistCurve replaces the generics with a version using the wrappedPoint
// interface, then update all the method signatures to also use wrappedPoint.
type nistCurve struct {
	newPoint func() wrappedPoint
	params   *CurveParams
}

//gopherjs:override-signature
func (curve *nistCurve) Params() *CurveParams {}

//gopherjs:override-signature
func (curve *nistCurve) IsOnCurve(x, y *big.Int) bool {}

//gopherjs:override-signature
func (curve *nistCurve) pointFromAffine(x, y *big.Int) (p wrappedPoint, err error) {}

//gopherjs:override-signature
func (curve *nistCurve) pointToAffine(p wrappedPoint) (x, y *big.Int) {}

//gopherjs:override-signature
func (curve *nistCurve) Add(x1, y1, x2, y2 *big.Int) (*big.Int, *big.Int) {}

//gopherjs:override-signature
func (curve *nistCurve) Double(x1, y1 *big.Int) (*big.Int, *big.Int) {}

//gopherjs:override-signature
func (curve *nistCurve) normalizeScalar(scalar []byte) []byte {}

//gopherjs:override-signature
func (curve *nistCurve) ScalarMult(Bx, By *big.Int, scalar []byte) (*big.Int, *big.Int) {}

//gopherjs:override-signature
func (curve *nistCurve) ScalarBaseMult(scalar []byte) (*big.Int, *big.Int) {}

//gopherjs:override-signature
func (curve *nistCurve) CombinedMult(Px, Py *big.Int, s1, s2 []byte) (x, y *big.Int) {}

//gopherjs:override-signature
func (curve *nistCurve) Unmarshal(data []byte) (x, y *big.Int) {}

//gopherjs:override-signature
func (curve *nistCurve) UnmarshalCompressed(data []byte) (x, y *big.Int) {}

var p224 = &nistCurve{
	newPoint: newP224WrappedPoint,
}

type p224Wrapper struct {
	point *nistec.P224Point
}

func wrapP224(point *nistec.P224Point) wrappedPoint {
	return p224Wrapper{point: point}
}

func newP224WrappedPoint() wrappedPoint {
	return wrapP224(nistec.NewP224Point())
}

func (w p224Wrapper) Bytes() []byte {
	return w.point.Bytes()
}

func (w p224Wrapper) SetBytes(b []byte) (wrappedPoint, error) {
	p, err := w.point.SetBytes(b)
	return wrapP224(p), err
}

func (w p224Wrapper) Add(w1, w2 wrappedPoint) wrappedPoint {
	return wrapP224(w.point.Add(w1.(p224Wrapper).point, w2.(p224Wrapper).point))
}

func (w p224Wrapper) Double(w1 wrappedPoint) wrappedPoint {
	return wrapP224(w.point.Double(w1.(p224Wrapper).point))
}

func (w p224Wrapper) ScalarMult(w1 wrappedPoint, scalar []byte) (wrappedPoint, error) {
	p, err := w.point.ScalarMult(w1.(p224Wrapper).point, scalar)
	return wrapP224(p), err
}

func (w p224Wrapper) ScalarBaseMult(scalar []byte) (wrappedPoint, error) {
	p, err := w.point.ScalarBaseMult(scalar)
	return wrapP224(p), err
}

type p256Curve struct {
	nistCurve
}

var p256 = &p256Curve{
	nistCurve: nistCurve{
		newPoint: newP256WrappedPoint,
	},
}

type p256Wrapper struct {
	point *nistec.P256Point
}

func wrapP256(point *nistec.P256Point) wrappedPoint {
	return p256Wrapper{point: point}
}

func newP256WrappedPoint() wrappedPoint {
	return wrapP256(nistec.NewP256Point())
}

func (w p256Wrapper) Bytes() []byte {
	return w.point.Bytes()
}

func (w p256Wrapper) SetBytes(b []byte) (wrappedPoint, error) {
	p, err := w.point.SetBytes(b)
	return wrapP256(p), err
}

func (w p256Wrapper) Add(w1, w2 wrappedPoint) wrappedPoint {
	return wrapP256(w.point.Add(w1.(p256Wrapper).point, w2.(p256Wrapper).point))
}

func (w p256Wrapper) Double(w1 wrappedPoint) wrappedPoint {
	return wrapP256(w.point.Double(w1.(p256Wrapper).point))
}

func (w p256Wrapper) ScalarMult(w1 wrappedPoint, scalar []byte) (wrappedPoint, error) {
	p, err := w.point.ScalarMult(w1.(p256Wrapper).point, scalar)
	return wrapP256(p), err
}

func (w p256Wrapper) ScalarBaseMult(scalar []byte) (wrappedPoint, error) {
	p, err := w.point.ScalarBaseMult(scalar)
	return wrapP256(p), err
}

var p521 = &nistCurve{
	newPoint: newP521WrappedPoint,
}

type p521Wrapper struct {
	point *nistec.P521Point
}

func wrapP521(point *nistec.P521Point) wrappedPoint {
	return p521Wrapper{point: point}
}

func newP521WrappedPoint() wrappedPoint {
	return wrapP521(nistec.NewP521Point())
}

func (w p521Wrapper) Bytes() []byte {
	return w.point.Bytes()
}

func (w p521Wrapper) SetBytes(b []byte) (wrappedPoint, error) {
	p, err := w.point.SetBytes(b)
	return wrapP521(p), err
}

func (w p521Wrapper) Add(w1, w2 wrappedPoint) wrappedPoint {
	return wrapP521(w.point.Add(w1.(p521Wrapper).point, w2.(p521Wrapper).point))
}

func (w p521Wrapper) Double(w1 wrappedPoint) wrappedPoint {
	return wrapP521(w.point.Double(w1.(p521Wrapper).point))
}

func (w p521Wrapper) ScalarMult(w1 wrappedPoint, scalar []byte) (wrappedPoint, error) {
	p, err := w.point.ScalarMult(w1.(p521Wrapper).point, scalar)
	return wrapP521(p), err
}

func (w p521Wrapper) ScalarBaseMult(scalar []byte) (wrappedPoint, error) {
	p, err := w.point.ScalarBaseMult(scalar)
	return wrapP521(p), err
}

var p384 = &nistCurve{
	newPoint: newP384WrappedPoint,
}

type p384Wrapper struct {
	point *nistec.P384Point
}

func wrapP384(point *nistec.P384Point) wrappedPoint {
	return p384Wrapper{point: point}
}

func newP384WrappedPoint() wrappedPoint {
	return wrapP384(nistec.NewP384Point())
}

func (w p384Wrapper) Bytes() []byte {
	return w.point.Bytes()
}

func (w p384Wrapper) SetBytes(b []byte) (wrappedPoint, error) {
	p, err := w.point.SetBytes(b)
	return wrapP384(p), err
}

func (w p384Wrapper) Add(w1, w2 wrappedPoint) wrappedPoint {
	return wrapP384(w.point.Add(w1.(p384Wrapper).point, w2.(p384Wrapper).point))
}

func (w p384Wrapper) Double(w1 wrappedPoint) wrappedPoint {
	return wrapP384(w.point.Double(w1.(p384Wrapper).point))
}

func (w p384Wrapper) ScalarMult(w1 wrappedPoint, scalar []byte) (wrappedPoint, error) {
	p, err := w.point.ScalarMult(w1.(p384Wrapper).point, scalar)
	return wrapP384(p), err
}

func (w p384Wrapper) ScalarBaseMult(scalar []byte) (wrappedPoint, error) {
	p, err := w.point.ScalarBaseMult(scalar)
	return wrapP384(p), err
}
