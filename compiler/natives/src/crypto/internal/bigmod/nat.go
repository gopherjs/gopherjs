//go:build js
// +build js

package bigmod

import (
	"errors"
	"math/big"
	"math/bits"
)

// This overrides the use of `uint` in the original code with `uint32`
// for GopherJS since `uint32` will handle the multiplications correctly
// with $imul where as the `uint` uses `*` which will not truncate causing
// a loss of information in the LSBs.
//
// This is not a great approach since it requires copying large amount
// of the mathematics just to change a handful of `uint` to `uint32`.
// Therefore this solution isn't very future proof. We should consider
// a better approach to handle this change.

type choice uint32

func ctSelect(on choice, x, y uint32) uint32 {
	mask := -uint32(on)
	return y ^ (mask & (y ^ x))
}

func ctEq(x, y uint32) choice {
	_, c1 := bits.Sub32(x, y, 0)
	_, c2 := bits.Sub32(y, x, 0)
	return not(choice(c1 | c2))
}

func ctGeq(x, y uint32) choice {
	_, carry := bits.Sub32(x, y, 0)
	return not(choice(carry))
}

type Nat struct {
	limbs []uint32
}

func NewNat() *Nat {
	limbs := make([]uint32, 0, preallocLimbs)
	return &Nat{limbs}
}

func (x *Nat) expand(n int) *Nat {
	if len(x.limbs) > n {
		panic("bigmod: internal error: shrinking nat")
	}
	if cap(x.limbs) < n {
		newLimbs := make([]uint32, n)
		copy(newLimbs, x.limbs)
		x.limbs = newLimbs
		return x
	}
	extraLimbs := x.limbs[len(x.limbs):n]
	for i := range extraLimbs {
		extraLimbs[i] = 0
	}
	x.limbs = x.limbs[:n]
	return x
}

func (x *Nat) reset(n int) *Nat {
	if cap(x.limbs) < n {
		x.limbs = make([]uint32, n)
		return x
	}
	for i := range x.limbs {
		x.limbs[i] = 0
	}
	x.limbs = x.limbs[:n]
	return x
}

func (x *Nat) setBig(n *big.Int) *Nat {
	requiredLimbs := (n.BitLen() + _W - 1) / _W
	x.reset(requiredLimbs)

	outI := 0
	shift := 0
	limbs := n.Bits()
	for i := range limbs {
		xi := uint32(limbs[i])
		x.limbs[outI] |= (xi << shift) & _MASK
		outI++
		if outI == requiredLimbs {
			return x
		}
		x.limbs[outI] = xi >> (_W - shift)
		shift++
		if shift == _W {
			shift = 0
			outI++
		}
	}
	return x
}

func (x *Nat) setBytes(b []byte, m *Modulus) error {
	outI := 0
	shift := 0
	x.resetFor(m)
	for i := len(b) - 1; i >= 0; i-- {
		bi := b[i]
		x.limbs[outI] |= uint32(bi) << shift
		shift += 8
		if shift >= _W {
			shift -= _W
			x.limbs[outI] &= _MASK
			overflow := bi >> (8 - shift)
			outI++
			if outI >= len(x.limbs) {
				if overflow > 0 || i > 0 {
					return errors.New("input overflows the modulus")
				}
				break
			}
			x.limbs[outI] = uint32(overflow)
		}
	}
	return nil
}

func (x *Nat) cmpGeq(y *Nat) choice {
	size := len(x.limbs)
	xLimbs := x.limbs[:size]
	yLimbs := y.limbs[:size]

	var c uint32
	for i := 0; i < size; i++ {
		c = (xLimbs[i] - yLimbs[i] - c) >> _W
	}
	return not(choice(c))
}

//gopherjs:override-signature
func (x *Nat) add(on choice, y *Nat) (c uint32)

//gopherjs:override-signature
func (x *Nat) sub(on choice, y *Nat) (c uint32)

type Modulus struct {
	nat     *Nat
	leading int
	m0inv   uint32
	rr      *Nat
}

//gopherjs:override-signature
func minusInverseModW(x uint32) uint32

//gopherjs:override-signature
func bitLen(n uint32) int

func (x *Nat) shiftIn(y uint32, m *Modulus) *Nat {
	d := NewNat().resetFor(m)

	size := len(m.nat.limbs)
	xLimbs := x.limbs[:size]
	dLimbs := d.limbs[:size]
	mLimbs := m.nat.limbs[:size]

	needSubtraction := no
	for i := _W - 1; i >= 0; i-- {
		carry := (y >> i) & 1
		var borrow uint32
		for i := 0; i < size; i++ {
			l := ctSelect(needSubtraction, dLimbs[i], xLimbs[i])

			res := l<<1 + carry
			xLimbs[i] = res & _MASK
			carry = res >> _W

			res = xLimbs[i] - mLimbs[i] - borrow
			dLimbs[i] = res & _MASK
			borrow = res >> _W
		}
		needSubtraction = ctEq(carry, borrow)
	}
	return x.assign(needSubtraction, d)
}

func (x *Nat) Add(y *Nat, m *Modulus) *Nat {
	overflow := x.add(yes, y)
	underflow := not(x.cmpGeq(m.nat))

	needSubtraction := ctEq(overflow, uint32(underflow))

	x.sub(needSubtraction, m.nat)
	return x
}

func (d *Nat) montgomeryMul(a *Nat, b *Nat, m *Modulus) *Nat {
	d.resetFor(m)
	if len(a.limbs) != len(m.nat.limbs) || len(b.limbs) != len(m.nat.limbs) {
		panic("bigmod: invalid montgomeryMul input")
	}

	//GOPHERJS: Update montgomeryLoop to montgomeryLoopGeneric
	overflow := montgomeryLoop(d.limbs, a.limbs, b.limbs, m.nat.limbs, m.m0inv)
	underflow := not(d.cmpGeq(m.nat))
	needSubtraction := ctEq(overflow, uint32(underflow))
	d.sub(needSubtraction, m.nat)

	return d
}

func montgomeryLoopGeneric(d, a, b, m []uint32, m0inv uint32) (overflow uint32) {
	// Eliminate bounds checks in the loop.
	size := len(d)
	a = a[:size]
	b = b[:size]
	m = m[:size]

	for _, ai := range a {
		// This is an unrolled iteration of the loop below with j = 0.
		hi, lo := bits.Mul32(ai, b[0])
		z_lo, c := bits.Add32(d[0], lo, 0)
		f := (z_lo * m0inv) & _MASK // (d[0] + a[i] * b[0]) * m0inv
		z_hi, _ := bits.Add32(0, hi, c)
		hi, lo = bits.Mul32(f, m[0])
		z_lo, c = bits.Add32(z_lo, lo, 0)
		z_hi, _ = bits.Add32(z_hi, hi, c)
		carry := z_hi<<1 | z_lo>>_W

		for j := 1; j < size; j++ {
			// z = d[j] + a[i] * b[j] + f * m[j] + carry <= 2^(2W+1) - 2^(W+1) + 2^W
			hi, lo := bits.Mul32(ai, b[j])
			z_lo, c := bits.Add32(d[j], lo, 0)
			z_hi, _ := bits.Add32(0, hi, c)
			hi, lo = bits.Mul32(f, m[j])
			z_lo, c = bits.Add32(z_lo, lo, 0)
			z_hi, _ = bits.Add32(z_hi, hi, c)
			z_lo, c = bits.Add32(z_lo, carry, 0)
			z_hi, _ = bits.Add32(z_hi, 0, c)
			d[j-1] = z_lo & _MASK
			carry = z_hi<<1 | z_lo>>_W // carry <= 2^(W+1) - 2
		}

		z := overflow + carry // z <= 2^(W+1) - 1
		d[size-1] = z & _MASK
		overflow = z >> _W // overflow <= 1
	}
	return
}

func (out *Nat) Exp(x *Nat, e []byte, m *Modulus) *Nat {
	table := [(1 << 4) - 1]*Nat{
		NewNat(), NewNat(), NewNat(), NewNat(), NewNat(),
		NewNat(), NewNat(), NewNat(), NewNat(), NewNat(),
		NewNat(), NewNat(), NewNat(), NewNat(), NewNat(),
	}
	table[0].set(x).montgomeryRepresentation(m)
	for i := 1; i < len(table); i++ {
		table[i].montgomeryMul(table[i-1], table[0], m)
	}

	out.resetFor(m)
	out.limbs[0] = 1
	out.montgomeryRepresentation(m)
	t0 := NewNat().ExpandFor(m)
	t1 := NewNat().ExpandFor(m)
	for _, b := range e {
		for _, j := range []int{4, 0} {
			t1.montgomeryMul(out, out, m)
			out.montgomeryMul(t1, t1, m)
			t1.montgomeryMul(out, out, m)
			out.montgomeryMul(t1, t1, m)

			k := uint32((b >> j) & 0b1111)
			for i := range table {
				t0.assign(ctEq(k, uint32(i+1)), table[i])
			}

			t1.montgomeryMul(out, t0, m)
			out.assign(not(ctEq(k, 0)), t1)
		}
	}

	return out.montgomeryReduction(m)
}

//gopherjs:override-signature
func montgomeryLoop(d, a, b, m []uint32, m0inv uint32) uint32
