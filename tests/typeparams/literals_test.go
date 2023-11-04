package typeparams_test

import (
	"fmt"
	"testing"

	"golang.org/x/exp/constraints"
)

func intLit[T constraints.Integer]() T {
	var i T = 1
	return i
}

func runeLit[T rune]() T {
	var r T = 'a'
	return r
}

func floatLit[T constraints.Float]() T {
	var f T = 1.1
	return f
}

func complexLit[T constraints.Complex]() T {
	var c T = 1 + 2i
	return c
}

func complexLit2[T constraints.Complex]() T {
	var c T = 1
	return c
}

func strLit[T string]() T {
	var s T = "abc"
	return s
}

func boolLit[T bool]() T {
	var b T = true
	return b
}

func nilLit[T *int]() T {
	var p T = nil
	return p
}

func TestLiterals(t *testing.T) {
	tests := []struct {
		got  any
		want any
	}{{
		got:  intLit[int32](),
		want: int32(1),
	}, {
		got:  intLit[uint32](),
		want: uint32(1),
	}, {
		got:  intLit[int64](),
		want: int64(1),
	}, {
		got:  intLit[uint64](),
		want: uint64(1),
	}, {
		got:  runeLit[rune](),
		want: 'a',
	}, {
		got:  floatLit[float32](),
		want: float32(1.1),
	}, {
		got:  floatLit[float64](),
		want: float64(1.1),
	}, {
		got:  complexLit[complex64](),
		want: complex64(1 + 2i),
	}, {
		got:  complexLit[complex128](),
		want: complex128(1 + 2i),
	}, {
		got:  complexLit2[complex128](),
		want: complex128(1),
	}, {
		got:  strLit[string](),
		want: "abc",
	}, {
		got:  boolLit[bool](),
		want: true,
	}, {
		got:  nilLit[*int](),
		want: (*int)(nil),
	}}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%T/%v", test.want, test.want), func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("Got: %v. Want: %v.", test.got, test.want)
			}
		})
	}
}
