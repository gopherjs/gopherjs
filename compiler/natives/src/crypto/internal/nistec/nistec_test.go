//go:build js
// +build js

package nistec_test

import (
	"crypto/elliptic"
	"crypto/internal/nistec"
	"testing"
)

func TestAllocations(t *testing.T) {
	t.Skip("testing.AllocsPerRun not supported in GopherJS")
}

//gopherjs:purge
type nistPoint[T any] interface{}

func TestEquivalents(t *testing.T) {
	t.Run("P224", func(t *testing.T) {
		testEquivalents(t, nistec.NewP224WrappedPoint, elliptic.P224())
	})
	t.Run("P256", func(t *testing.T) {
		testEquivalents(t, nistec.NewP256WrappedPoint, elliptic.P256())
	})
	t.Run("P384", func(t *testing.T) {
		testEquivalents(t, nistec.NewP384WrappedPoint, elliptic.P384())
	})
	t.Run("P521", func(t *testing.T) {
		testEquivalents(t, nistec.NewP521WrappedPoint, elliptic.P521())
	})
}

//gopherjs:override-signature
func testEquivalents(t *testing.T, newPoint func() nistec.WrappedPoint, c elliptic.Curve)

func TestScalarMult(t *testing.T) {
	t.Run("P224", func(t *testing.T) {
		testScalarMult(t, nistec.NewP224WrappedPoint, elliptic.P224())
	})
	t.Run("P256", func(t *testing.T) {
		testScalarMult(t, nistec.NewP256WrappedPoint, elliptic.P256())
	})
	t.Run("P384", func(t *testing.T) {
		testScalarMult(t, nistec.NewP384WrappedPoint, elliptic.P384())
	})
	t.Run("P521", func(t *testing.T) {
		testScalarMult(t, nistec.NewP521WrappedPoint, elliptic.P521())
	})
}

//gopherjs:override-signature
func testScalarMult(t *testing.T, newPoint func() nistec.WrappedPoint, c elliptic.Curve)

func BenchmarkScalarMult(b *testing.B) {
	b.Run("P224", func(b *testing.B) {
		benchmarkScalarMult(b, nistec.NewP224WrappedPoint().SetGenerator(), 28)
	})
	b.Run("P256", func(b *testing.B) {
		benchmarkScalarMult(b, nistec.NewP256WrappedPoint().SetGenerator(), 32)
	})
	b.Run("P384", func(b *testing.B) {
		benchmarkScalarMult(b, nistec.NewP384WrappedPoint().SetGenerator(), 48)
	})
	b.Run("P521", func(b *testing.B) {
		benchmarkScalarMult(b, nistec.NewP521WrappedPoint().SetGenerator(), 66)
	})
}

//gopherjs:override-signature
func benchmarkScalarMult(b *testing.B, p nistec.WrappedPoint, scalarSize int)

func BenchmarkScalarBaseMult(b *testing.B) {
	b.Run("P224", func(b *testing.B) {
		benchmarkScalarBaseMult(b, nistec.NewP224WrappedPoint().SetGenerator(), 28)
	})
	b.Run("P256", func(b *testing.B) {
		benchmarkScalarBaseMult(b, nistec.NewP256WrappedPoint().SetGenerator(), 32)
	})
	b.Run("P384", func(b *testing.B) {
		benchmarkScalarBaseMult(b, nistec.NewP384WrappedPoint().SetGenerator(), 48)
	})
	b.Run("P521", func(b *testing.B) {
		benchmarkScalarBaseMult(b, nistec.NewP521WrappedPoint().SetGenerator(), 66)
	})
}

//gopherjs:override-signature
func benchmarkScalarBaseMult(b *testing.B, p nistec.WrappedPoint, scalarSize int)
