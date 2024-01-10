//go:build js
// +build js

package elliptic

import (
	"crypto/internal/nistec"
	"reflect"
	"sync"
)

// The following code changes in this file are to make p224, p256,
// p521, and p384 still function correctly without this generic struct
//
//gopherjs:purge for go1.19 without generics
type nistPoint[T any] interface{}

type reflectivePointCaller struct {
	Bytes          func(recv any) []byte
	SetBytes       func(recv any, b []byte) (any, error)
	Add            func(recv, p1, p2 any) any
	Double         func(recv, p1 any) any
	ScalarMult     func(recv, p1 any, scalar []byte) (any, error)
	ScalarBaseMult func(recv any, scalar []byte) (any, error)
}

var pointCallerInit sync.Once
var pointCallerSingleton *reflectivePointCaller

func initPointCallerInstance() {
	pointCallerInit.Do(func() {
		wrapFunc := func(fName string, fPtr any) {
			fn := reflect.ValueOf(fPtr).Elem()
			caller := func(args []reflect.Value) []reflect.Value {
				return args[0].MethodByName(fName).Call(args[1:])
			}
			v := reflect.MakeFunc(fn.Type(), caller)
			fn.Set(v)
		}

		rpc := &reflectivePointCaller{}
		wrapFunc(`Bytes`, &rpc.Bytes)
		wrapFunc(`SetBytes`, &rpc.SetBytes)
		wrapFunc(`Add`, &rpc.Add)
		wrapFunc(`Double`, &rpc.Double)
		wrapFunc(`ScalarMult`, &rpc.ScalarMult)
		wrapFunc(`ScalarBaseMult`, &rpc.ScalarBaseMult)
		pointCallerSingleton = rpc
	})
}

type wrappedPoint struct {
	point any
}

func newWrappedPoint(point any) *wrappedPoint {
	initPointCallerInstance()
	return &wrappedPoint{point: point}
}

func newPointWrapper(newPoint func() any) func() *wrappedPoint {
	return func() *wrappedPoint {
		return newWrappedPoint(newPoint())
	}
}

func (w *wrappedPoint) Bytes() []byte {
	return pointCallerSingleton.Bytes(w.point)
}

func (w *wrappedPoint) SetBytes(b []byte) (*wrappedPoint, error) {
	p, err := pointCallerSingleton.SetBytes(w.point, b)
	return newWrappedPoint(p), err
}

func (w *wrappedPoint) Add(w1, w2 *wrappedPoint) *wrappedPoint {
	return newWrappedPoint(pointCallerSingleton.Add(w.point, w1.point, w2.point))
}

func (w *wrappedPoint) Double(w1 *wrappedPoint) *wrappedPoint {
	return newWrappedPoint(pointCallerSingleton.Double(w.point, w1.point))
}

func (w *wrappedPoint) ScalarMult(w1 *wrappedPoint, scalar []byte) (*wrappedPoint, error) {
	p, err := pointCallerSingleton.ScalarMult(w.point, w1.point, scalar)
	return newWrappedPoint(p), err
}

func (w *wrappedPoint) ScalarBaseMult(scalar []byte) (*wrappedPoint, error) {
	p, err := pointCallerSingleton.ScalarBaseMult(w.point, scalar)
	return newWrappedPoint(p), err
}

type nistCurve struct {
	newPoint func() *wrappedPoint
	params   *CurveParams
}

func newNistCurve(newPoint func() any) *nistCurve {
	return &nistCurve{newPoint: newPointWrapper(newPoint)}
}

var p224 = newNistCurve(func() any { return nistec.NewP224Point() })

type p256Curve struct {
	nistCurve
}

var p256 = &p256Curve{
	nistCurve: *newNistCurve(func() any { return nistec.NewP256Point() }),
}

var p521 = newNistCurve(func() any { return nistec.NewP521Point() })

var p384 = newNistCurve(func() any { return nistec.NewP384Point() })
