//go:build js
// +build js

package testing

//gopherjs:purge for go1.19 without generics
func genericHelper[G any](t *T, msg string) {}

//gopherjs:purge for go1.19 without generics
var genericIntHelper = genericHelper[int]

//gopherjs:purge for go1.19 without generics (uses genericHelper)
func testHelper(t *T) {}
