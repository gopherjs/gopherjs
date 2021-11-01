//go:build js
// +build js

package bits

type _err string

func (e _err) Error() string {
	return string(e)
}

// RuntimeError implements runtime.Error.
func (e _err) RuntimeError() {
}

var (
	overflowError error = _err("runtime error: integer overflow")
	divideError   error = _err("runtime error: integer divide by zero")
)

func Mul32(x, y uint32) (hi, lo uint32) {
	// Avoid slow 64-bit integers for better performance. Adapted from Mul64().
	const mask16 = 1<<16 - 1
	x0 := x & mask16
	x1 := x >> 16
	y0 := y & mask16
	y1 := y >> 16
	w0 := x0 * y0
	t := x1*y0 + w0>>16
	w1 := t & mask16
	w2 := t >> 16
	w1 += x0 * y1
	hi = x1*y1 + w2 + w1>>16
	lo = x * y
	return
}

func Add32(x, y, carry uint32) (sum, carryOut uint32) {
	// Avoid slow 64-bit integers for better performance. Adapted from Add64().
	sum = x + y + carry
	carryOut = ((x & y) | ((x | y) &^ sum)) >> 31
	return
}

func Div32(hi, lo, y uint32) (quo, rem uint32) {
	// Avoid slow 64-bit integers for better performance. Adapted from Div64().
	const (
		two16  = 1 << 16
		mask16 = two16 - 1
	)
	if y == 0 {
		panic(divideError)
	}
	if y <= hi {
		panic(overflowError)
	}

	s := uint(LeadingZeros32(y))
	y <<= s

	yn1 := y >> 16
	yn0 := y & mask16
	un16 := hi<<s | lo>>(32-s)
	un10 := lo << s
	un1 := un10 >> 16
	un0 := un10 & mask16
	q1 := un16 / yn1
	rhat := un16 - q1*yn1

	for q1 >= two16 || q1*yn0 > two16*rhat+un1 {
		q1--
		rhat += yn1
		if rhat >= two16 {
			break
		}
	}

	un21 := un16*two16 + un1 - q1*y
	q0 := un21 / yn1
	rhat = un21 - q0*yn1

	for q0 >= two16 || q0*yn0 > two16*rhat+un0 {
		q0--
		rhat += yn1
		if rhat >= two16 {
			break
		}
	}

	return q1*two16 + q0, (un21*two16 + un0 - q0*y) >> s
}

func Rem32(hi, lo, y uint32) uint32 {
	// We scale down hi so that hi < y, then use Div32 to compute the
	// rem with the guarantee that it won't panic on quotient overflow.
	// Given that
	//   hi ≡ hi%y    (mod y)
	// we have
	//   hi<<64 + lo ≡ (hi%y)<<64 + lo    (mod y)
	_, rem := Div32(hi%y, lo, y)
	return rem
}
