//go:build js
// +build js

package elliptic

// these are overwritten with a placeholder to pass the type
// check that is performed in the original elliptic.go
var p224, p256, p384, p521 interface {
	Curve
	unmarshaler
}

//gopherjs:purge for go1.19 without generics
type p256Curve struct{}

//gopherjs:purge for go1.19 without generics
func initP224() {}

//gopherjs:purge for go1.19 without generics
func initP256() {}

//gopherjs:purge for go1.19 without generics
func initP384() {}

//gopherjs:purge for go1.19 without generics
func initP521() {}

//gopherjs:purge for go1.19 without generics
type nistCurve[Point nistPoint[Point]] struct{}

//gopherjs:purge for go1.19 without generics
type nistPoint[T any] interface{}
