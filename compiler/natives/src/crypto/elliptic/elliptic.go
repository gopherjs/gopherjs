//go:build js
// +build js

package elliptic

//gopherjs:purge for go1.19 without generics
func initAll() {}

//gopherjs:purge for go1.19 without generics
func P224() Curve { return nil }

//gopherjs:purge for go1.19 without generics
func P256() Curve { return nil }

//gopherjs:purge for go1.19 without generics
func P384() Curve { return nil }

//gopherjs:purge for go1.19 without generics
func P521() Curve { return nil }
