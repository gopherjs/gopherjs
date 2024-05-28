//go:build js
// +build js

package nistec

// temporarily replacement of `nistPoint[T any]` for go1.20 without generics.
type WrappedPoint interface {
	SetGenerator() WrappedPoint
	Bytes() []byte
	BytesX() ([]byte, error)
	SetBytes(b []byte) (WrappedPoint, error)
	Add(w1, w2 WrappedPoint) WrappedPoint
	Double(w1 WrappedPoint) WrappedPoint
	ScalarMult(w1 WrappedPoint, scalar []byte) (WrappedPoint, error)
	ScalarBaseMult(scalar []byte) (WrappedPoint, error)
}

type p224Wrapper struct {
	point *P224Point
}

func wrapP224(point *P224Point) WrappedPoint {
	return p224Wrapper{point: point}
}

func NewP224WrappedPoint() WrappedPoint {
	return wrapP224(NewP224Point())
}

func (w p224Wrapper) SetGenerator() WrappedPoint {
	return wrapP224(w.point.SetGenerator())
}

func (w p224Wrapper) Bytes() []byte {
	return w.point.Bytes()
}

func (w p224Wrapper) BytesX() ([]byte, error) {
	return w.point.BytesX()
}

func (w p224Wrapper) SetBytes(b []byte) (WrappedPoint, error) {
	p, err := w.point.SetBytes(b)
	return wrapP224(p), err
}

func (w p224Wrapper) Add(w1, w2 WrappedPoint) WrappedPoint {
	return wrapP224(w.point.Add(w1.(p224Wrapper).point, w2.(p224Wrapper).point))
}

func (w p224Wrapper) Double(w1 WrappedPoint) WrappedPoint {
	return wrapP224(w.point.Double(w1.(p224Wrapper).point))
}

func (w p224Wrapper) ScalarMult(w1 WrappedPoint, scalar []byte) (WrappedPoint, error) {
	p, err := w.point.ScalarMult(w1.(p224Wrapper).point, scalar)
	return wrapP224(p), err
}

func (w p224Wrapper) ScalarBaseMult(scalar []byte) (WrappedPoint, error) {
	p, err := w.point.ScalarBaseMult(scalar)
	return wrapP224(p), err
}

type p256Wrapper struct {
	point *P256Point
}

func wrapP256(point *P256Point) WrappedPoint {
	return p256Wrapper{point: point}
}

func NewP256WrappedPoint() WrappedPoint {
	return wrapP256(NewP256Point())
}

func (w p256Wrapper) SetGenerator() WrappedPoint {
	return wrapP256(w.point.SetGenerator())
}

func (w p256Wrapper) Bytes() []byte {
	return w.point.Bytes()
}

func (w p256Wrapper) BytesX() ([]byte, error) {
	return w.point.BytesX()
}

func (w p256Wrapper) SetBytes(b []byte) (WrappedPoint, error) {
	p, err := w.point.SetBytes(b)
	return wrapP256(p), err
}

func (w p256Wrapper) Add(w1, w2 WrappedPoint) WrappedPoint {
	return wrapP256(w.point.Add(w1.(p256Wrapper).point, w2.(p256Wrapper).point))
}

func (w p256Wrapper) Double(w1 WrappedPoint) WrappedPoint {
	return wrapP256(w.point.Double(w1.(p256Wrapper).point))
}

func (w p256Wrapper) ScalarMult(w1 WrappedPoint, scalar []byte) (WrappedPoint, error) {
	p, err := w.point.ScalarMult(w1.(p256Wrapper).point, scalar)
	return wrapP256(p), err
}

func (w p256Wrapper) ScalarBaseMult(scalar []byte) (WrappedPoint, error) {
	p, err := w.point.ScalarBaseMult(scalar)
	return wrapP256(p), err
}

type p521Wrapper struct {
	point *P521Point
}

func wrapP521(point *P521Point) WrappedPoint {
	return p521Wrapper{point: point}
}

func NewP521WrappedPoint() WrappedPoint {
	return wrapP521(NewP521Point())
}

func (w p521Wrapper) SetGenerator() WrappedPoint {
	return wrapP521(w.point.SetGenerator())
}

func (w p521Wrapper) Bytes() []byte {
	return w.point.Bytes()
}

func (w p521Wrapper) BytesX() ([]byte, error) {
	return w.point.BytesX()
}

func (w p521Wrapper) SetBytes(b []byte) (WrappedPoint, error) {
	p, err := w.point.SetBytes(b)
	return wrapP521(p), err
}

func (w p521Wrapper) Add(w1, w2 WrappedPoint) WrappedPoint {
	return wrapP521(w.point.Add(w1.(p521Wrapper).point, w2.(p521Wrapper).point))
}

func (w p521Wrapper) Double(w1 WrappedPoint) WrappedPoint {
	return wrapP521(w.point.Double(w1.(p521Wrapper).point))
}

func (w p521Wrapper) ScalarMult(w1 WrappedPoint, scalar []byte) (WrappedPoint, error) {
	p, err := w.point.ScalarMult(w1.(p521Wrapper).point, scalar)
	return wrapP521(p), err
}

func (w p521Wrapper) ScalarBaseMult(scalar []byte) (WrappedPoint, error) {
	p, err := w.point.ScalarBaseMult(scalar)
	return wrapP521(p), err
}

type p384Wrapper struct {
	point *P384Point
}

func wrapP384(point *P384Point) WrappedPoint {
	return p384Wrapper{point: point}
}

func NewP384WrappedPoint() WrappedPoint {
	return wrapP384(NewP384Point())
}

func (w p384Wrapper) SetGenerator() WrappedPoint {
	return wrapP384(w.point.SetGenerator())
}

func (w p384Wrapper) Bytes() []byte {
	return w.point.Bytes()
}

func (w p384Wrapper) BytesX() ([]byte, error) {
	return w.point.BytesX()
}

func (w p384Wrapper) SetBytes(b []byte) (WrappedPoint, error) {
	p, err := w.point.SetBytes(b)
	return wrapP384(p), err
}

func (w p384Wrapper) Add(w1, w2 WrappedPoint) WrappedPoint {
	return wrapP384(w.point.Add(w1.(p384Wrapper).point, w2.(p384Wrapper).point))
}

func (w p384Wrapper) Double(w1 WrappedPoint) WrappedPoint {
	return wrapP384(w.point.Double(w1.(p384Wrapper).point))
}

func (w p384Wrapper) ScalarMult(w1 WrappedPoint, scalar []byte) (WrappedPoint, error) {
	p, err := w.point.ScalarMult(w1.(p384Wrapper).point, scalar)
	return wrapP384(p), err
}

func (w p384Wrapper) ScalarBaseMult(scalar []byte) (WrappedPoint, error) {
	p, err := w.point.ScalarBaseMult(scalar)
	return wrapP384(p), err
}
