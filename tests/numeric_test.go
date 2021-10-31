package tests

import (
	"math/rand"
	"runtime"
	"testing"
	"testing/quick"
)

// naiveMul64 performs 64-bit multiplication without using the multiplication
// operation and can be used to test correctness of the compiler's multiplication
// implementation.
func naiveMul64(x, y uint64) uint64 {
	var z uint64 = 0
	for i := 0; i < 64; i++ {
		mask := uint64(1) << i
		if y&mask > 0 {
			z += x << i
		}
	}
	return z
}

func TestMul64(t *testing.T) {
	cfg := &quick.Config{
		MaxCountScale: 10000,
		Rand:          rand.New(rand.NewSource(0x5EED)), // Fixed seed for reproducibility.
	}
	if testing.Short() {
		cfg.MaxCountScale = 1000
	}

	t.Run("unsigned", func(t *testing.T) {
		err := quick.CheckEqual(
			func(x, y uint64) uint64 { return x * y },
			naiveMul64,
			cfg)
		if err != nil {
			t.Error(err)
		}
	})
	t.Run("signed", func(t *testing.T) {
		// GopherJS represents 64-bit signed integers in a two-complement form,
		// so bitwise multiplication looks identical for signed and unsigned integers
		// and we can reuse naiveMul64() as a reference implementation for both with
		// appropriate type conversions.
		err := quick.CheckEqual(
			func(x, y int64) int64 { return x * y },
			func(x, y int64) int64 { return int64(naiveMul64(uint64(x), uint64(y))) },
			cfg)
		if err != nil {
			t.Error(err)
		}
	})
}

func BenchmarkMul64(b *testing.B) {
	// Prepare a randomized set of multipliers to make sure the benchmark doesn't
	// get too specific for a single value. The trade-off is that the cost of
	// loading from an array gets mixed into the result, but it is good enough for
	// relative comparisons.
	r := rand.New(rand.NewSource(0x5EED))
	const size = 1024
	xU := [size]uint64{}
	yU := [size]uint64{}
	xS := [size]int64{}
	yS := [size]int64{}
	for i := 0; i < size; i++ {
		xU[i] = r.Uint64()
		yU[i] = r.Uint64()
		xS[i] = r.Int63() | (r.Int63n(2) << 63)
		yS[i] = r.Int63() | (r.Int63n(2) << 63)
	}

	b.Run("noop", func(b *testing.B) {
		// This benchmark allows to gauge the cost of array load operations without
		// the multiplications.
		for i := 0; i < b.N; i++ {
			runtime.KeepAlive(yU[i%size])
			runtime.KeepAlive(xU[i%size])
		}
	})
	b.Run("unsigned", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			z := xU[i%size] * yU[i%size]
			runtime.KeepAlive(z)
		}
	})
	b.Run("signed", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			z := xS[i%size] * yS[i%size]
			runtime.KeepAlive(z)
		}
	})
}
