//go:build js

package math

import "github.com/gopherjs/gopherjs/js"

var (
	math           = js.Global.Get("Math")
	_zero  float64 = 0
	posInf         = 1 / _zero
	negInf         = -1 / _zero
)

// Usually, NaN can be obtained in JavaScript with `0 / 0` operation. However,
// in V8, `0 / _zero` yields a bitwise-different value of NaN compared to the
// default NaN or `0 / 0`. Unfortunately, Go compiler forbids division by zero,
// so we have to get this value from prelude.
var nan = js.Global.Get("$NaN").Float()

//gopherjs:replace
func Acos(x float64) float64 {
	return math.Call("acos", x).Float()
}

//gopherjs:replace
func Acosh(x float64) float64 {
	return math.Call("acosh", x).Float()
}

//gopherjs:replace
func Asin(x float64) float64 {
	return math.Call("asin", x).Float()
}

//gopherjs:replace
func Asinh(x float64) float64 {
	return math.Call("asinh", x).Float()
}

//gopherjs:replace
func Atan(x float64) float64 {
	return math.Call("atan", x).Float()
}

//gopherjs:replace
func Atanh(x float64) float64 {
	return math.Call("atanh", x).Float()
}

//gopherjs:replace
func Atan2(y, x float64) float64 {
	return math.Call("atan2", y, x).Float()
}

//gopherjs:replace
func Cbrt(x float64) float64 {
	return math.Call("cbrt", x).Float()
}

//gopherjs:replace
func Ceil(x float64) float64 {
	return math.Call("ceil", x).Float()
}

//gopherjs:replace
func Copysign(x, y float64) float64 {
	if (x < 0 || 1/x == negInf) != (y < 0 || 1/y == negInf) {
		return -x
	}
	return x
}

//gopherjs:replace
func Cos(x float64) float64 {
	return math.Call("cos", x).Float()
}

//gopherjs:replace
func Cosh(x float64) float64 {
	return math.Call("cosh", x).Float()
}

//gopherjs:replace
func Erf(x float64) float64 {
	return erf(x)
}

//gopherjs:replace
func Erfc(x float64) float64 {
	return erfc(x)
}

//gopherjs:replace
func Exp(x float64) float64 {
	return math.Call("exp", x).Float()
}

//gopherjs:replace
func Exp2(x float64) float64 {
	return math.Call("pow", 2, x).Float()
}

//gopherjs:replace
func Expm1(x float64) float64 {
	return expm1(x)
}

//gopherjs:replace
func Floor(x float64) float64 {
	return math.Call("floor", x).Float()
}

//gopherjs:replace
func Frexp(f float64) (frac float64, exp int) {
	return frexp(f)
}

//gopherjs:replace
func Hypot(p, q float64) float64 {
	return hypot(p, q)
}

//gopherjs:replace
func Inf(sign int) float64 {
	switch {
	case sign >= 0:
		return posInf
	default:
		return negInf
	}
}

//gopherjs:replace
func IsInf(f float64, sign int) bool {
	if f == posInf {
		return sign >= 0
	}
	if f == negInf {
		return sign <= 0
	}
	return false
}

//gopherjs:replace
func IsNaN(f float64) (is bool) {
	return f != f
}

//gopherjs:replace
func Ldexp(frac float64, exp int) float64 {
	if -1024 < exp && exp < 1024 { // Use Math.pow for small exp values where it's viable. For performance.
		if frac == 0 {
			return frac
		}
		return frac * math.Call("pow", 2, exp).Float()
	}
	return ldexp(frac, exp)
}

//gopherjs:replace
func Log(x float64) float64 {
	if x != x { // workaround for optimizer bug in V8, remove at some point
		return nan
	}
	return math.Call("log", x).Float()
}

//gopherjs:replace
func Log10(x float64) float64 {
	return log10(x)
}

//gopherjs:replace
func Log1p(x float64) float64 {
	return log1p(x)
}

//gopherjs:replace
func Log2(x float64) float64 {
	return log2(x)
}

//gopherjs:replace
func Max(x, y float64) float64 {
	return max(x, y)
}

//gopherjs:replace
func Min(x, y float64) float64 {
	return min(x, y)
}

//gopherjs:replace
func Mod(x, y float64) float64 {
	return js.Global.Call("$mod", x, y).Float()
}

//gopherjs:replace
func Modf(f float64) (float64, float64) {
	if f == posInf || f == negInf {
		return f, nan
	}
	if 1/f == negInf {
		return f, f
	}
	frac := Mod(f, 1)
	return f - frac, frac
}

//gopherjs:replace
func NaN() float64 {
	return nan
}

//gopherjs:replace
func Pow(x, y float64) float64 {
	if x == 1 || (x == -1 && (y == posInf || y == negInf)) {
		return 1
	}
	return math.Call("pow", x, y).Float()
}

//gopherjs:replace
func Remainder(x, y float64) float64 {
	return remainder(x, y)
}

//gopherjs:replace
func Signbit(x float64) bool {
	return x < 0 || 1/x == negInf
}

//gopherjs:replace
func Sin(x float64) float64 {
	return math.Call("sin", x).Float()
}

//gopherjs:replace
func Sinh(x float64) float64 {
	return math.Call("sinh", x).Float()
}

//gopherjs:replace
func Sincos(x float64) (sin, cos float64) {
	return Sin(x), Cos(x)
}

//gopherjs:replace
func Sqrt(x float64) float64 {
	return math.Call("sqrt", x).Float()
}

//gopherjs:replace
func Tan(x float64) float64 {
	return math.Call("tan", x).Float()
}

//gopherjs:replace
func Tanh(x float64) float64 {
	return math.Call("tanh", x).Float()
}

//gopherjs:replace
func Trunc(x float64) float64 {
	if x == posInf || x == negInf || x != x || 1/x == negInf {
		return x
	}
	return Copysign(float64(int(x)), x)
}

var buf struct {
	uint32array  [2]uint32
	float32array [2]float32
	float64array [1]float64
}

func init() {
	ab := js.Global.Get("ArrayBuffer").New(8)
	js.InternalObject(buf).Set("uint32array", js.Global.Get("Uint32Array").New(ab))
	js.InternalObject(buf).Set("float32array", js.Global.Get("Float32Array").New(ab))
	js.InternalObject(buf).Set("float64array", js.Global.Get("Float64Array").New(ab))
}

//gopherjs:replace
func Float32bits(f float32) uint32 {
	buf.float32array[0] = f
	return buf.uint32array[0]
}

//gopherjs:replace
func Float32frombits(b uint32) float32 {
	buf.uint32array[0] = b
	return buf.float32array[0]
}

//gopherjs:replace
func Float64bits(f float64) uint64 {
	buf.float64array[0] = f
	return uint64(buf.uint32array[1])<<32 + uint64(buf.uint32array[0])
}

//gopherjs:replace
func Float64frombits(b uint64) float64 {
	buf.uint32array[0] = uint32(b)
	buf.uint32array[1] = uint32(b >> 32)
	return buf.float64array[0]
}
