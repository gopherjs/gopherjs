//go:build js
// +build js

package elliptic

import (
	"crypto/internal/nistec"
	"reflect"
)

// The following code changes in this file are to make p224, p256,
// p521, and p384 still function correctly without this generic struct
//
//gopherjs:purge for go1.19 without generics
type nistCurve[Point nistPoint[Point]] struct{}

//gopherjs:purge for go1.19 without generics
type nistPoint[T any] interface{}

type wrappedPoint struct {
	point              any
	funcBytes          func() []byte
	funcSetBytes       func(b []byte) (any, error)
	funcAdd            func(p1, p2 any) any
	funcDouble         func(p1 any) any
	funcScalarMult     func(p1 any, scalar []byte) (any, error)
	funcScalarBaseMult func(scalar []byte) (any, error)
}

func newWrappedPoint(point any) *wrappedPoint {
	return &wrappedPoint{point: point}
}

func newPointWrapper(newPoint func() any) func() *wrappedPoint {
	return func() *wrappedPoint {
		return newWrappedPoint(newPoint())
	}
}

func (w *wrappedPoint) wrapFunc(fName string, fPtr any) {
	fn := reflect.ValueOf(fPtr).Elem()
	target := reflect.ValueOf(w.point).MethodByName(fName).Call
	v := reflect.MakeFunc(fn.Type(), target)
	fn.Set(v)
}

func (w *wrappedPoint) Bytes() []byte {
	if w.funcBytes == nil {
		w.wrapFunc(`Bytes`, &w.funcBytes)
	}
	return w.funcBytes()
}

func (w *wrappedPoint) SetBytes(b []byte) (*wrappedPoint, error) {
	if w.funcSetBytes == nil {
		w.wrapFunc(`SetBytes`, &w.funcSetBytes)
	}
	p, err := w.funcSetBytes(b)
	return newWrappedPoint(p), err
}

func (w *wrappedPoint) Add(w1, w2 *wrappedPoint) *wrappedPoint {
	if w.funcAdd == nil {
		w.wrapFunc(`Add`, &w.funcAdd)
	}
	return newWrappedPoint(w.funcAdd(w1.point, w2.point))
}

func (w *wrappedPoint) Double(w1 *wrappedPoint) *wrappedPoint {
	if w.funcDouble == nil {
		w.wrapFunc(`Double`, &w.funcDouble)
	}
	return newWrappedPoint(w.funcDouble(w1.point))
}

func (w *wrappedPoint) ScalarMult(w1 *wrappedPoint, scalar []byte) (*wrappedPoint, error) {
	if w.funcScalarMult == nil {
		w.wrapFunc(`ScalarMult`, &w.funcScalarMult)
	}
	p, err := w.funcScalarMult(w1.point, scalar)
	return newWrappedPoint(p), err
}

func (w *wrappedPoint) ScalarBaseMult(scalar []byte) (*wrappedPoint, error) {
	if w.funcScalarBaseMult == nil {
		w.wrapFunc(`ScalarBaseMult`, &w.funcScalarBaseMult)
	}
	p, err := w.funcScalarBaseMult(scalar)
	return newWrappedPoint(p), err
}

type wrappingNistCurve struct {
	newPoint func() *wrappedPoint
	params   *CurveParams
}

var p224 = &wrappingNistCurve{
	newPoint: newPointWrapper(func() any {
		return nistec.NewP224Point()
	}),
}

type p256Curve struct {
	nistCurve
}

var p256 = &p256Curve{
	nistCurve: wrappingNistCurve{
		newPoint: newPointWrapper(func() any {
			return nistec.NewP256Point()
		}),
	},
}

var p521 = &wrappingNistCurve{
	newPoint: newPointWrapper(func() any {
		return nistec.NewP521Point()
	}),
}

var p384 = &wrappingNistCurve{
	newPoint: newPointWrapper(func() any {
		return nistec.NewP384Point()
	}),
}
