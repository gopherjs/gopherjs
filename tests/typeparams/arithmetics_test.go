package typeparams_test

import (
	"fmt"
	"go/token"
	"testing"

	"golang.org/x/exp/constraints"
)

type testCaseI interface {
	Run(t *testing.T)
	String() string
}

type testCase[T comparable] struct {
	op     func(x, y T) T
	opName token.Token
	x      T
	y      T
	want   T
}

func (tc *testCase[T]) Run(t *testing.T) {
	got := tc.op(tc.x, tc.y)
	if got != tc.want {
		t.Errorf("Got: %v %v %v = %v. Want: %v.", tc.x, tc.opName, tc.y, got, tc.want)
	}
}

func (tc *testCase[T]) String() string {
	return fmt.Sprintf("%T/%v%v%v", tc.x, tc.x, tc.opName, tc.y)
}

type addable interface {
	constraints.Integer | constraints.Float | constraints.Complex | string
}

func add[T addable](x, y T) T {
	return x + y
}

func addTC[T addable](x, y, want T) *testCase[T] {
	return &testCase[T]{
		op:     add[T],
		opName: token.ADD,
		x:      x,
		y:      y,
		want:   want,
	}
}

func TestAdd(t *testing.T) {
	tests := []testCaseI{
		addTC[int](1, 2, 3),
		addTC[uint](1, 2, 3),
		addTC[uintptr](1, 2, 3),
		addTC[int8](1, 2, 3),
		addTC[int16](1, 2, 3),
		addTC[int32](1, 2, 3),
		addTC[uint8](1, 2, 3),
		addTC[uint16](1, 2, 3),
		addTC[uint32](1, 2, 3),
		addTC[int8](127, 2, -127), // Overflow.
		addTC[uint8](255, 2, 1),   // Overflow.
		addTC[float32](1.5, 1.1, 2.6),
		addTC[float64](1.5, 1.1, 2.6),
		addTC[int64](0x00000030FFFFFFFF, 0x0000000100000002, 0x0000003200000001),
		addTC[uint64](0x00000030FFFFFFFF, 0x0000000100000002, 0x0000003200000001),
		addTC[string]("abc", "def", "abcdef"),
		addTC[complex64](1+2i, 3+4i, 4+6i),
		addTC[complex128](1+2i, 3+4i, 4+6i),
	}

	for _, test := range tests {
		t.Run(test.String(), test.Run)
	}
}

type subtractable interface {
	constraints.Integer | constraints.Float | constraints.Complex
}

func subtract[T subtractable](x, y T) T {
	return x - y
}

func subTC[T subtractable](x, y, want T) *testCase[T] {
	return &testCase[T]{
		op:     subtract[T],
		opName: token.SUB,
		x:      x,
		y:      y,
		want:   want,
	}
}

func TestSubtract(t *testing.T) {
	tests := []testCaseI{
		subTC[int](3, 1, 2),
		subTC[uint](3, 1, 2),
		subTC[uintptr](3, 1, 2),
		subTC[int8](3, 1, 2),
		subTC[int16](3, 1, 2),
		subTC[int32](3, 1, 2),
		subTC[uint8](3, 1, 2),
		subTC[uint16](3, 1, 2),
		subTC[uint32](3, 1, 2),
		subTC[int8](-127, 2, 127), // Overflow.
		subTC[uint8](1, 2, 255),   // Overflow.
		subTC[float32](2.5, 1.4, 1.1),
		subTC[float64](2.5, 1.4, 1.1),
		subTC[int64](0x0000003200000001, 0x0000000100000002, 0x00000030FFFFFFFF),
		subTC[uint64](0x0000003200000001, 0x0000000100000002, 0x00000030FFFFFFFF),
		subTC[complex64](10+11i, 2+1i, 8+10i),
		subTC[complex128](10+11i, 2+1i, 8+10i),
	}

	for _, test := range tests {
		t.Run(test.String(), test.Run)
	}
}
