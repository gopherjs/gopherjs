//go:build js
// +build js

package bigmod

import (
	"math/big"
	"math/rand"
	"reflect"
	"testing"
)

func (*Nat) Generate(r *rand.Rand, size int) reflect.Value {
	limbs := make([]uint32, size)
	for i := 0; i < size; i++ {
		limbs[i] = uint32(r.Uint64()) & ((1 << _W) - 2)
	}
	return reflect.ValueOf(&Nat{limbs})
}

func testMontgomeryRoundtrip(a *Nat) bool {
	one := &Nat{make([]uint32, len(a.limbs))}
	one.limbs[0] = 1
	aPlusOne := new(big.Int).SetBytes(natBytes(a))
	aPlusOne.Add(aPlusOne, big.NewInt(1))
	m := NewModulusFromBig(aPlusOne)
	monty := new(Nat).set(a)
	monty.montgomeryRepresentation(m)
	aAgain := new(Nat).set(monty)
	aAgain.montgomeryMul(monty, one, m)
	return a.Equal(aAgain) == 1
}

func TestShiftIn(t *testing.T) {
	t.Skip("examples are only valid in 64 bit")
}

func TestExpand(t *testing.T) {
	sliced := []uint32{1, 2, 3, 4}
	examples := []struct {
		in  []uint32
		n   int
		out []uint32
	}{{
		[]uint32{1, 2},
		4,
		[]uint32{1, 2, 0, 0},
	}, {
		sliced[:2],
		4,
		[]uint32{1, 2, 0, 0},
	}, {
		[]uint32{1, 2},
		2,
		[]uint32{1, 2},
	}}

	for i, tt := range examples {
		got := (&Nat{tt.in}).expand(tt.n)
		if len(got.limbs) != len(tt.out) || got.Equal(&Nat{tt.out}) != 1 {
			t.Errorf("%d: got %x, expected %x", i, got, tt.out)
		}
	}
}

func TestModSub(t *testing.T) {
	m := modulusFromBytes([]byte{13})
	x := &Nat{[]uint32{6}}
	y := &Nat{[]uint32{7}}
	x.Sub(y, m)
	expected := &Nat{[]uint32{12}}
	if x.Equal(expected) != 1 {
		t.Errorf("%+v != %+v", x, expected)
	}
	x.Sub(y, m)
	expected = &Nat{[]uint32{5}}
	if x.Equal(expected) != 1 {
		t.Errorf("%+v != %+v", x, expected)
	}
}

func TestModAdd(t *testing.T) {
	m := modulusFromBytes([]byte{13})
	x := &Nat{[]uint32{6}}
	y := &Nat{[]uint32{7}}
	x.Add(y, m)
	expected := &Nat{[]uint32{0}}
	if x.Equal(expected) != 1 {
		t.Errorf("%+v != %+v", x, expected)
	}
	x.Add(y, m)
	expected = &Nat{[]uint32{7}}
	if x.Equal(expected) != 1 {
		t.Errorf("%+v != %+v", x, expected)
	}
}

func TestExp(t *testing.T) {
	m := modulusFromBytes([]byte{13})
	x := &Nat{[]uint32{3}}
	out := &Nat{[]uint32{0}}
	out.Exp(x, []byte{12}, m)
	expected := &Nat{[]uint32{1}}
	if out.Equal(expected) != 1 {
		t.Errorf("%+v != %+v", out, expected)
	}
}

func makeBenchmarkValue() *Nat {
	x := make([]uint32, 32)
	for i := 0; i < 32; i++ {
		x[i] = _MASK - 1
	}
	return &Nat{limbs: x}
}
