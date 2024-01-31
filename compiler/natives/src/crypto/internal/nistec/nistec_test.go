//go:build js
// +build js

package nistec_test

import (
	"testing"

	"crypto/elliptic"
	"crypto/internal/nistec"
)

func TestAllocations(t *testing.T) {
	t.Skip("testing.AllocsPerRun not supported in GopherJS")
}

//gopherjs:purge
type nistPoint[T any] interface{}

func TestEquivalents(t *testing.T) {
	t.Run("P224", func(t *testing.T) {
		testEquivalents(t, nistec.NewP224WrappedPoint, nistec.NewP224WrappedGenerator, elliptic.P224())
	})
	t.Run("P256", func(t *testing.T) {
		testEquivalents(t, nistec.NewP256WrappedPoint, nistec.NewP256WrappedGenerator, elliptic.P256())
	})
	t.Run("P384", func(t *testing.T) {
		testEquivalents(t, nistec.NewP384WrappedPoint, nistec.NewP384WrappedGenerator, elliptic.P384())
	})
	t.Run("P521", func(t *testing.T) {
		testEquivalents(t, nistec.NewP521WrappedPoint, nistec.NewP521WrappedGenerator, elliptic.P521())
	})
}

//gopherjs:override-signature
func testEquivalents(t *testing.T, newPoint, newGenerator func() nistec.WrappedPoint, c elliptic.Curve)

func TestScalarMult(t *testing.T) {
	t.Run("P224", func(t *testing.T) {
		testScalarMult(t, nistec.NewP224WrappedPoint, nistec.NewP224WrappedGenerator, elliptic.P224())
	})
	t.Run("P256", func(t *testing.T) {
		testScalarMult(t, nistec.NewP256WrappedPoint, nistec.NewP256WrappedGenerator, elliptic.P256())
	})
	t.Run("P384", func(t *testing.T) {
		testScalarMult(t, nistec.NewP384WrappedPoint, nistec.NewP384WrappedGenerator, elliptic.P384())
	})
	t.Run("P521", func(t *testing.T) {
		testScalarMult(t, nistec.NewP521WrappedPoint, nistec.NewP521WrappedGenerator, elliptic.P521())
	})
}

//gopherjs:override-signature
func testScalarMult(t *testing.T, newPoint, newGenerator func() nistec.WrappedPoint, c elliptic.Curve)

func BenchmarkScalarMult(b *testing.B) {
	b.Run("P224", func(b *testing.B) {
		benchmarkScalarMult(b, nistec.NewP224WrappedGenerator(), 28)
	})
	b.Run("P256", func(b *testing.B) {
		benchmarkScalarMult(b, nistec.NewP256WrappedGenerator(), 32)
	})
	b.Run("P384", func(b *testing.B) {
		benchmarkScalarMult(b, nistec.NewP384WrappedGenerator(), 48)
	})
	b.Run("P521", func(b *testing.B) {
		benchmarkScalarMult(b, nistec.NewP521WrappedGenerator(), 66)
	})
}

//gopherjs:override-signature
func benchmarkScalarMult(b *testing.B, p nistec.WrappedPoint, scalarSize int)

func BenchmarkScalarBaseMult(b *testing.B) {
	b.Run("P224", func(b *testing.B) {
		benchmarkScalarBaseMult(b, nistec.NewP224WrappedGenerator(), 28)
	})
	b.Run("P256", func(b *testing.B) {
		benchmarkScalarBaseMult(b, nistec.NewP256WrappedGenerator(), 32)
	})
	b.Run("P384", func(b *testing.B) {
		benchmarkScalarBaseMult(b, nistec.NewP384WrappedGenerator(), 48)
	})
	b.Run("P521", func(b *testing.B) {
		benchmarkScalarBaseMult(b, nistec.NewP521WrappedGenerator(), 66)
	})
}

//gopherjs:override-signature
func benchmarkScalarBaseMult(b *testing.B, p nistec.WrappedPoint, scalarSize int)
