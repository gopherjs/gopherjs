//go:build js && !wasm

package tests

import (
	"fmt"
	"internal/chacha8rand"
	"testing"
	_ "unsafe"

	"github.com/gopherjs/gopherjs/js"
)

// jsOrigRand64 is the original (`fastrand64`) equivalent to `runtime.rand`.
func jsOrigRand64() uint64 {
	return uint64(jsOrigRand32())<<32 | uint64(jsOrigRand32())
}

func jsOrigRand32() uint32 {
	return uint32(js.Global.Get("Math").Call("random").Float() * (1<<32 - 1))
}

// jsAltRand64 is an alternate version of the original verison leveraging `js.MakeUint64`.
func jsAltRand64() uint64 {
	return js.MakeUint64(jsAltRand32f(), jsAltRand32f())
}

func jsAltRand32f() float64 {
	math := js.Global.Get(`Math`)
	return math.Call("floor", math.Call("random").Float()*(1<<32-1)).Float()
}

var goChacha8 chacha8rand.State

// goRand64 is the upsteam implementation of `runtime.rand`.
// based on https://cs.opensource.google/go/go/+/refs/tags/go1.22.12:src/runtime/rand.go;l=127
func goRand64() uint64 {
	// This is simplified because JS doesn't care about nosplit and
	// will not prempt this run even without the pseudo locking.
	c := &goChacha8
	for {
		x, ok := c.Next()
		if ok {
			return x
		}
		c.Refill()
	}
}

// This had to be overwritten because it contains a conversion `*[16][4]uint32` => `*[16][2]uint64`.
// Changed the `*[16][4]uint32` parameter on upstream to `*[32]uint64` so each `b[x][y]` became `b[x<<1|y]`.
//
// based on https://cs.opensource.google/go/go/+/refs/tags/go1.22.12:src/internal/chacha8rand/chacha8_generic.go
func chacha8_setup(seed *[4]uint64, b *[32]uint64, counter uint32) {
	// Constants; same as in ChaCha20: "expand 32-byte k"
	b[0] = 0x61707865_61707865
	b[1] = 0x61707865_61707865

	b[2] = 0x3320646e_3320646e
	b[3] = 0x3320646e_3320646e

	b[4] = 0x79622d32_79622d32
	b[5] = 0x79622d32_79622d32

	b[6] = 0x6b206574_6b206574
	b[7] = 0x6b206574_6b206574

	// Seed values.
	var x64 uint64
	var x uint32

	x = uint32(seed[0])
	x64 = uint64(x)<<32 | uint64(x)
	b[8] = x64
	b[9] = x64

	x = uint32(seed[0] >> 32)
	x64 = uint64(x)<<32 | uint64(x)
	b[10] = x64
	b[11] = x64

	x = uint32(seed[1])
	x64 = uint64(x)<<32 | uint64(x)
	b[12] = x64
	b[13] = x64

	x = uint32(seed[1] >> 32)
	x64 = uint64(x)<<32 | uint64(x)
	b[14] = x64
	b[15] = x64

	x = uint32(seed[2])
	x64 = uint64(x)<<32 | uint64(x)
	b[16] = x64
	b[17] = x64

	x = uint32(seed[2] >> 32)
	x64 = uint64(x)<<32 | uint64(x)
	b[18] = x64
	b[19] = x64

	x = uint32(seed[3])
	x64 = uint64(x)<<32 | uint64(x)
	b[20] = x64
	b[21] = x64

	x = uint32(seed[3] >> 32)
	x64 = uint64(x)<<32 | uint64(x)
	b[22] = x64
	b[23] = x64

	// Counters.
	b[24] = uint64(counter+0) | uint64(counter+1)<<32
	b[25] = uint64(counter+2) | uint64(counter+3)<<32

	// Zeros.
	b[26] = 0
	b[27] = 0
	b[28] = 0
	b[29] = 0

	b[30] = 0
	b[31] = 0
}

// This is to override the `internal/chacha8rand/chacha8_stub.s` implementation
// that calls block_generic and instead this is block_generic's implementation.
//
//go:linkname chacha8_block internal/chacha8rand.block
func chacha8_block(seed *[4]uint64, buf *[32]uint64, counter uint32) {
	chacha8_setup(seed, buf, counter)

	halfBlock := func(b []uint64) {
		// Load block i from b[*][i] into local variables.
		b0hi, b0lo := js.Uint64High(b[0]), js.Uint64Low(b[0])
		b1hi, b1lo := js.Uint64High(b[1]), js.Uint64Low(b[1])
		b2hi, b2lo := js.Uint64High(b[2]), js.Uint64Low(b[2])
		b3hi, b3lo := js.Uint64High(b[3]), js.Uint64Low(b[3])
		b4hi, b4lo := js.Uint64High(b[4]), js.Uint64Low(b[4])
		b5hi, b5lo := js.Uint64High(b[5]), js.Uint64Low(b[5])
		b6hi, b6lo := js.Uint64High(b[6]), js.Uint64Low(b[6])
		b7hi, b7lo := js.Uint64High(b[7]), js.Uint64Low(b[7])
		b8hi, b8lo := js.Uint64High(b[8]), js.Uint64Low(b[8])
		b9hi, b9lo := js.Uint64High(b[9]), js.Uint64Low(b[9])
		b10hi, b10lo := js.Uint64High(b[10]), js.Uint64Low(b[10])
		b11hi, b11lo := js.Uint64High(b[11]), js.Uint64Low(b[11])
		b12hi, b12lo := js.Uint64High(b[12]), js.Uint64Low(b[12])
		b13hi, b13lo := js.Uint64High(b[13]), js.Uint64Low(b[13])
		b14hi, b14lo := js.Uint64High(b[14]), js.Uint64Low(b[14])
		b15hi, b15lo := js.Uint64High(b[15]), js.Uint64Low(b[15])

		// 4 iterations of eight quarter-rounds each is 8 rounds
		for round := 0; round < 4; round++ {
			b0hi, b4hi, b8hi, b12hi = chacha8_qr(b0hi, b4hi, b8hi, b12hi)
			b1hi, b5hi, b9hi, b13hi = chacha8_qr(b1hi, b5hi, b9hi, b13hi)
			b2hi, b6hi, b10hi, b14hi = chacha8_qr(b2hi, b6hi, b10hi, b14hi)
			b3hi, b7hi, b11hi, b15hi = chacha8_qr(b3hi, b7hi, b11hi, b15hi)

			b0hi, b5hi, b10hi, b15hi = chacha8_qr(b0hi, b5hi, b10hi, b15hi)
			b1hi, b6hi, b11hi, b12hi = chacha8_qr(b1hi, b6hi, b11hi, b12hi)
			b2hi, b7hi, b8hi, b13hi = chacha8_qr(b2hi, b7hi, b8hi, b13hi)
			b3hi, b4hi, b9hi, b14hi = chacha8_qr(b3hi, b4hi, b9hi, b14hi)

			b0lo, b4lo, b8lo, b12lo = chacha8_qr(b0lo, b4lo, b8lo, b12lo)
			b1lo, b5lo, b9lo, b13lo = chacha8_qr(b1lo, b5lo, b9lo, b13lo)
			b2lo, b6lo, b10lo, b14lo = chacha8_qr(b2lo, b6lo, b10lo, b14lo)
			b3lo, b7lo, b11lo, b15lo = chacha8_qr(b3lo, b7lo, b11lo, b15lo)

			b0lo, b5lo, b10lo, b15lo = chacha8_qr(b0lo, b5lo, b10lo, b15lo)
			b1lo, b6lo, b11lo, b12lo = chacha8_qr(b1lo, b6lo, b11lo, b12lo)
			b2lo, b7lo, b8lo, b13lo = chacha8_qr(b2lo, b7lo, b8lo, b13lo)
			b3lo, b4lo, b9lo, b14lo = chacha8_qr(b3lo, b4lo, b9lo, b14lo)
		}

		// Store block i back into b[*][i].
		// Add b4..b11 back to the original key material,
		// like in ChaCha20, to avoid trivial invertibility.
		// There is no entropy in b0..b3 and b12..b15
		// so we can skip the additions and save some time.
		b[0] = js.MakeUint64(float64(b0hi), float64(b0lo))
		b[1] = js.MakeUint64(float64(b1hi), float64(b1lo))
		b[2] = js.MakeUint64(float64(b2hi), float64(b2lo))
		b[3] = js.MakeUint64(float64(b3hi), float64(b3lo))
		b[4] += js.MakeUint64(float64(b4hi), float64(b4lo))
		b[5] += js.MakeUint64(float64(b5hi), float64(b5lo))
		b[6] += js.MakeUint64(float64(b6hi), float64(b6lo))
		b[7] += js.MakeUint64(float64(b7hi), float64(b7lo))
		b[8] += js.MakeUint64(float64(b8hi), float64(b8lo))
		b[9] += js.MakeUint64(float64(b9hi), float64(b9lo))
		b[10] += js.MakeUint64(float64(b10hi), float64(b10lo))
		b[11] += js.MakeUint64(float64(b11hi), float64(b11lo))
		b[12] = js.MakeUint64(float64(b12hi), float64(b12lo))
		b[13] = js.MakeUint64(float64(b13hi), float64(b13lo))
		b[14] = js.MakeUint64(float64(b14hi), float64(b14lo))
		b[15] = js.MakeUint64(float64(b15hi), float64(b15lo))
	}
	halfBlock(buf[:16])
	halfBlock(buf[16:])
}

//go:linkname chacha8_qr internal/chacha8rand.qr
func chacha8_qr(a, b, c, d uint32) (_a, _b, _c, _d uint32)

// Run with `go install . && gopherjs test --bench BenchmarkRuntimeRand --run=^$ ./tests`
func BenchmarkRuntimeRand(b *testing.B) {
	runBenchRuntimeRand(b, `jsOrigRand64`, jsOrigRand64)
	runBenchRuntimeRand(b, `jsAltRand64`, jsAltRand64)
	runBenchRuntimeRand(b, `goRand64`, goRand64)
}

func runBenchRuntimeRand(b *testing.B, name string, fn func() uint64) {
	b.Run(name, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = fn()
		}
	})
}

// Run with `go install . && gopherjs test --run=TestRuntimeRand ./tests`
func TestRuntimeRand(t *testing.T) {
	runTestRuntimeRand(t, `jsOrigRand64`, jsOrigRand64)
	runTestRuntimeRand(t, `jsAltRand64`, jsAltRand64)
	runTestRuntimeRand(t, `goRand64`, goRand64)
}

func runTestRuntimeRand(t *testing.T, name string, fn func() uint64) {
	const chunkBits = 8 // must evenly divide 64
	const chunkCount = 64 / chunkBits
	const bucketCount = 1 << chunkBits
	const chunkMask = bucketCount - 1
	const sampleCount = 1_000_000
	const mseThreshold = 0.0005 // allowed variance in distribution
	t.Run(name, func(t *testing.T) {
		buckets := make([]uint32, bucketCount)
		for i := 0; i < sampleCount; i++ {
			sample := fn()
			// fmt.Printf("%d) sample: %016X\n", i, sample)
			for j := 0; j < chunkCount; j++ {
				bucketIndex := sample & chunkMask
				sample >>= chunkBits
				// fmt.Printf("\tremains: %016X, bucketIndex: %02X\n", sample, bucketIndex)
				buckets[bucketIndex]++
				if buckets[bucketIndex] == 0 {
					t.Errorf(`bucket %d rolled over`, bucketIndex)
				}
			}
		}
		// fmt.Printf("Buckets: %v\n", buckets)

		// calculate normalized mean squared error (mse)
		predicted := float64(sampleCount) * float64(chunkCount) / float64(bucketCount)
		// fmt.Printf("predicted = %f\n", predicted)
		errSqrSum := 0.0
		errMax := 0.0
		errMin := 2.0
		for _, bucket := range buckets {
			normErr := (float64(bucket) - float64(predicted)) / float64(predicted)
			errSqrSum += (normErr * normErr)
			errMax = max(errMax, normErr)
			errMin = min(errMin, normErr)
		}
		mse := errSqrSum / float64(bucketCount)
		fmt.Printf("MSE = %f\n", mse)
		fmt.Printf("err Max = %f\n", errMax)
		fmt.Printf("err Min = %f\n", errMin)
		if mse > mseThreshold {
			t.Errorf(`mse was greater than %f: %f`, mseThreshold, mse)
		}
	})
}
