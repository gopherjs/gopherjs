//go:build js
// +build js

package ecdsa

import "testing"

//gopherjs:override-signature
func testRandomPoint(t *testing.T, c *nistCurve)

//gopherjs:override-signature
func testHashToNat(t *testing.T, c *nistCurve)
