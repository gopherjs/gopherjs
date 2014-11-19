// +build js

package math

import (
	"github.com/gopherjs/gopherjs/js"
)

var math = js.Global.Get("Math")
var zero float64 = 0
var posInf = 1 / zero
var negInf = -1 / zero
var nan = 0 / zero

func init() {
	// avoid dead code elimination
	Float32bits(0)
	Float32frombits(0)
}

func Abs(x float64) float64 {
	return abs(x)
}

func Acos(x float64) float64 {
	return math.Call("acos", x).Float()
}

func Asin(x float64) float64 {
	return math.Call("asin", x).Float()
}

func Atan(x float64) float64 {
	return math.Call("atan", x).Float()
}

func Atan2(y, x float64) float64 {
	return math.Call("atan2", y, x).Float()
}

func Ceil(x float64) float64 {
	return math.Call("ceil", x).Float()
}

func Copysign(x, y float64) float64 {
	if (x < 0 || 1/x == negInf) != (y < 0 || 1/y == negInf) {
		return -x
	}
	return x
}

func Cos(x float64) float64 {
	// return math.Call("cos", x).Float() // not precise enough, see https://code.google.com/p/v8/issues/detail?id=3006
	return cos(x)
}

func Dim(x, y float64) float64 {
	return dim(x, y)
}

func Exp(x float64) float64 {
	return math.Call("exp", x).Float()
}

func Exp2(x float64) float64 {
	return math.Call("pow", 2, x).Float()
}

func Expm1(x float64) float64 {
	return expm1(x)
}

func Floor(x float64) float64 {
	return math.Call("floor", x).Float()
}

func Frexp(f float64) (frac float64, exp int) {
	return frexp(f)
}

func Hypot(p, q float64) float64 {
	return hypot(p, q)
}

func Inf(sign int) float64 {
	switch {
	case sign >= 0:
		return posInf
	default:
		return negInf
	}
}

func IsInf(f float64, sign int) bool {
	if f == posInf {
		return sign >= 0
	}
	if f == negInf {
		return sign <= 0
	}
	return false
}

func IsNaN(f float64) (is bool) {
	return f != f
}

func Ldexp(frac float64, exp int) float64 {
	if frac == 0 {
		return frac
	}
	if exp >= 1024 {
		return frac * math.Call("pow", 2, 1023).Float() * math.Call("pow", 2, exp-1023).Float()
	}
	if exp <= -1024 {
		return frac * math.Call("pow", 2, -1023).Float() * math.Call("pow", 2, exp+1023).Float()
	}
	return frac * math.Call("pow", 2, exp).Float()
}

func Log(x float64) float64 {
	if x != x { // workaround for optimizer bug in V8, remove at some point
		return nan
	}
	return math.Call("log", x).Float()
}

func Log10(x float64) float64 {
	return log10(x)
}

func Log1p(x float64) float64 {
	return log1p(x)
}

func Log2(x float64) float64 {
	return log2(x)
}

func Max(x, y float64) float64 {
	return max(x, y)
}

func Min(x, y float64) float64 {
	return min(x, y)
}

func Mod(x, y float64) float64 {
	return js.Global.Call("$mod", x, y).Float()
}

func Modf(f float64) (float64, float64) {
	if f == posInf || f == negInf {
		return f, nan
	}
	frac := Mod(f, 1)
	return f - frac, frac
}

func NaN() float64 {
	return nan
}

func Pow(x, y float64) float64 {
	if x == 1 || (x == -1 && (y == posInf || y == negInf)) {
		return 1
	}
	return math.Call("pow", x, y).Float()
}

func Remainder(x, y float64) float64 {
	return remainder(x, y)
}

func Signbit(x float64) bool {
	return x < 0 || 1/x == negInf
}

func Sin(x float64) float64 {
	// return math.Call("sin", x).Float() // not precise enough, see https://code.google.com/p/v8/issues/detail?id=3006
	return sin(x)
}

func Sincos(x float64) (sin, cos float64) {
	// return Sin(x), Cos(x) // not precise enough, see https://code.google.com/p/v8/issues/detail?id=3006
	return sincos(x)
}

func Sqrt(x float64) float64 {
	return math.Call("sqrt", x).Float()
}

func Tan(x float64) float64 {
	// return math.Call("tan", x).Float() // not precise enough, see https://code.google.com/p/v8/issues/detail?id=3006
	return tan(x)
}

func Trunc(x float64) float64 {
	if x == posInf || x == negInf || x != x || 1/x == negInf {
		return x
	}
	return float64(int(x))
}

func Float32bits(f float32) uint32 {
	if js.InternalObject(f) == js.InternalObject(0) {
		if js.InternalObject(1/f) == js.InternalObject(negInf) {
			return 1 << 31
		}
		return 0
	}
	if js.InternalObject(f) != js.InternalObject(f) { // NaN
		return 2143289344
	}

	s := uint32(0)
	if f < 0 {
		s = 1 << 31
		f = -f
	}

	e := uint32(127 + 23)
	for f >= 1<<24 {
		f /= 2
		e++
		if e == (1<<8)-1 {
			if f >= 1<<23 {
				f = float32(posInf)
			}
			break
		}
	}
	for f < 1<<23 {
		e--
		if e == 0 {
			break
		}
		f *= 2
	}

	r := js.Global.Call("$mod", f, 2).Float()
	if (r > 0.5 && r < 1) || r >= 1.5 { // round to nearest even
		f++
	}

	return s | uint32(e)<<23 | (uint32(f) &^ (1 << 23))
}

func Float32frombits(b uint32) float32 {
	s := float32(+1)
	if b&(1<<31) != 0 {
		s = -1
	}
	e := (b >> 23) & (1<<8 - 1)
	m := b & (1<<23 - 1)

	if e == (1<<8)-1 {
		if m == 0 {
			return s / 0 // Inf
		}
		return float32(nan)
	}
	if e != 0 {
		m += 1 << 23
	}
	if e == 0 {
		e = 1
	}

	return float32(Ldexp(float64(m), int(e)-127-23)) * s
}

func Float64bits(f float64) uint64 {
	if f == 0 {
		if 1/f == negInf {
			return 1 << 63
		}
		return 0
	}
	if f != f { // NaN
		return 9221120237041090561
	}

	s := uint64(0)
	if f < 0 {
		s = 1 << 63
		f = -f
	}

	e := uint32(1023 + 52)
	for f >= 1<<53 {
		f /= 2
		e++
		if e == (1<<11)-1 {
			break
		}
	}
	for f < 1<<52 {
		e--
		if e == 0 {
			break
		}
		f *= 2
	}

	return s | uint64(e)<<52 | (uint64(f) &^ (1 << 52))
}

func Float64frombits(b uint64) float64 {
	s := float64(+1)
	if b&(1<<63) != 0 {
		s = -1
	}
	e := (b >> 52) & (1<<11 - 1)
	m := b & (1<<52 - 1)

	if e == (1<<11)-1 {
		if m == 0 {
			return s / 0
		}
		return nan
	}
	if e != 0 {
		m += 1 << 52
	}
	if e == 0 {
		e = 1
	}

	return Ldexp(float64(m), int(e)-1023-52) * s
}
