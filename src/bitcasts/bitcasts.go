package main

import (
	"fmt"
	"math"
)

func main() {
	test32 := func(b uint32) {
		fmt.Println("---")
		f := math.Float32frombits(b)
		f2 := Float32frombits(b)
		b2 := Float32bits(f2)
		fmt.Printf("%f\n%f\n%032b\n%032b\n", f, f2, b, b2)
		fCorrect := (math.IsNaN(float64(f)) && math.IsNaN(float64(f2))) || f == f2
		if !fCorrect || b != b2 {
			panic("wrong")
		}
	}

	test64 := func(b uint64) {
		fmt.Println("---")
		f := math.Float64frombits(b)
		f2 := Float64frombits(b)
		b2 := Float64bits(f2)
		fmt.Printf("%f\n%f\n%064b\n%064b\n", f, f2, b, b2)
		fCorrect := (math.IsNaN(f) && math.IsNaN(f2)) || f == f2
		if !fCorrect || b != b2 {
			panic("wrong")
		}
	}

	test32(0)
	test32(1)
	test32(1109917696)
	test32(2139095039)
	test32(2139095040)
	test32(4286578688)
	test32(math.MaxUint32)

	test64(0)
	test64(1)
	test64(4631107791820423168)
	test64(9218868437227405311)
	test64(9218868437227405312)
	test64(18442240474082181120)
	test64(math.MaxUint64)

	fmt.Println("OK!")
}

func Exp2(x float64) float64 {
	return math.Exp2(x)
}

func Float32bits(f float32) uint32 {
	if f == 0 {
		return 0
	}
	if f != f { // NaN
		return 1<<32 - 1
	}

	s := uint32(0)
	if f < 0 {
		s = 1 << 31
		f = -f
	}

	e := uint32(127 + 23)
	for f >= 1<<24 {
		f /= 2
		if e == (1<<8)-1 {
			break
		}
		e += 1
	}
	for f < 1<<23 {
		e -= 1
		if e == 0 {
			break
		}
		f *= 2
	}

	return s | uint32(e)<<23 | (uint32(f) &^ (1 << 23))
}

func Float32frombits(b uint32) float32 {
	s := float32(+1)
	if b&(1<<31) != 0 {
		s = -1
	}
	e := int32(b>>23) & (1<<8 - 1)
	m := b & (1<<23 - 1)

	if e == (1<<8)-1 {
		if m == 0 {
			return s / 0 // Inf
		}
		return (s / 0) - (s / 0) // NaN
	}
	if e != 0 {
		m += 1 << 23
	}
	if e == 0 {
		e = 1
	}

	return float32(m) * float32(Exp2(float64(e-127-23))) * s
}

func Float64bits(f float64) uint64 {
	if f == 0 {
		return 0
	}
	if f != f { // NaN
		return 1<<64 - 1
	}

	s := uint64(0)
	if f < 0 {
		s = 1 << 63
		f = -f
	}

	e := uint32(1023 + 52)
	for f >= 1<<53 {
		f /= 2
		if e == (1<<11)-1 {
			break
		}
		e += 1
	}
	for f < 1<<52 {
		e -= 1
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
	e := int32(b>>52) & (1<<11 - 1)
	m := b & (1<<52 - 1)

	if e == (1<<11)-1 {
		if m == 0 {
			return s / 0
		}
		return (s / 0) - (s / 0) // NaN
	}
	if e != 0 {
		m += 1 << 52
	}
	if e == 0 {
		e = 1
	}

	return float64(m) * Exp2(float64(e-1023-52)) * s
}
