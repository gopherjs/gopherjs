package typeparams_test

import (
	"fmt"
	"go/token"
	"testing"

	"golang.org/x/exp/constraints"
)

type arithmetic interface {
	constraints.Integer | constraints.Float | constraints.Complex
}

type addable interface {
	arithmetic | string
}

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

func subtract[T arithmetic](x, y T) T {
	return x - y
}

func subTC[T arithmetic](x, y, want T) *testCase[T] {
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

func mul[T arithmetic](x, y T) T {
	return x * y
}

func mulTC[T arithmetic](x, y, want T) *testCase[T] {
	return &testCase[T]{
		op:     mul[T],
		opName: token.MUL,
		x:      x,
		y:      y,
		want:   want,
	}
}

func TestMul(t *testing.T) {
	tests := []testCaseI{
		mulTC[int](2, 3, 6),
		mulTC[uint](2, 3, 6),
		mulTC[uintptr](2, 3, 6),
		mulTC[int8](2, 3, 6),
		mulTC[int16](2, 3, 6),
		mulTC[int32](2, 3, 6),
		mulTC[uint8](2, 3, 6),
		mulTC[uint16](2, 3, 6),
		mulTC[uint32](2, 3, 6),
		mulTC[int8](127, 3, 125),  // Overflow.
		mulTC[uint8](250, 3, 238), // Overflow.
		mulTC[float32](2.5, 1.4, 3.5),
		mulTC[float64](2.5, 1.4, 3.5),
		mulTC[int64](0x0000003200000001, 0x0000000100000002, 0x0000006500000002),
		mulTC[uint64](0x0000003200000001, 0x0000000100000002, 0x0000006500000002),
		mulTC[complex64](1+2i, 3+4i, -5+10i),
		mulTC[complex128](1+2i, 3+4i, -5+10i),
	}

	for _, test := range tests {
		t.Run(test.String(), test.Run)
	}
}

func div[T arithmetic](x, y T) T {
	return x / y
}

func divTC[T arithmetic](x, y, want T) *testCase[T] {
	return &testCase[T]{
		op:     div[T],
		opName: token.QUO,
		x:      x,
		y:      y,
		want:   want,
	}
}

func TestDiv(t *testing.T) {
	tests := []testCaseI{
		divTC[int](7, 2, 3),
		divTC[uint](7, 2, 3),
		divTC[uintptr](7, 2, 3),
		divTC[int8](7, 2, 3),
		divTC[int16](7, 2, 3),
		divTC[int32](7, 2, 3),
		divTC[uint8](7, 2, 3),
		divTC[uint16](7, 2, 3),
		divTC[uint32](7, 2, 3),
		divTC[float32](3.5, 2.5, 1.4),
		divTC[float64](3.5, 2.5, 1.4),
		divTC[int64](0x0000006500000003, 0x0000003200000001, 2),
		divTC[uint64](0x0000006500000003, 0x0000003200000001, 2),
		divTC[complex64](-5+10i, 1+2i, 3+4i),
		divTC[complex128](-5+10i, 1+2i, 3+4i),
	}

	for _, test := range tests {
		t.Run(test.String(), test.Run)
	}
}

func rem[T constraints.Integer](x, y T) T {
	return x % y
}

func remTC[T constraints.Integer](x, y, want T) *testCase[T] {
	return &testCase[T]{
		op:     rem[T],
		opName: token.REM,
		x:      x,
		y:      y,
		want:   want,
	}
}

func TestRemainder(t *testing.T) {
	tests := []testCaseI{
		remTC[int](7, 2, 1),
		remTC[uint](7, 2, 1),
		remTC[uintptr](7, 2, 1),
		remTC[int8](7, 2, 1),
		remTC[int16](7, 2, 1),
		remTC[int32](7, 2, 1),
		remTC[uint8](7, 2, 1),
		remTC[uint16](7, 2, 1),
		remTC[uint32](7, 2, 1),
		remTC[int64](0x0000006500000003, 0x0000003200000001, 0x100000001),
		remTC[uint64](0x0000006500000003, 0x0000003200000001, 0x100000001),
	}

	for _, test := range tests {
		t.Run(test.String(), test.Run)
	}
}

func and[T constraints.Integer](x, y T) T {
	return x & y
}

func andTC[T constraints.Integer](x, y, want T) *testCase[T] {
	return &testCase[T]{
		op:     and[T],
		opName: token.AND,
		x:      x,
		y:      y,
		want:   want,
	}
}

func TestBitwiseAnd(t *testing.T) {
	tests := []testCaseI{
		andTC[int](0x0011, 0x0101, 0x0001),
		andTC[uint](0x0011, 0x0101, 0x0001),
		andTC[uintptr](0x0011, 0x0101, 0x0001),
		andTC[int8](0x11, 0x01, 0x01),
		andTC[int16](0x0011, 0x0101, 0x0001),
		andTC[int32](0x0011, 0x0101, 0x0001),
		andTC[uint8](0x11, 0x01, 0x01),
		andTC[uint16](0x0011, 0x0101, 0x0001),
		andTC[uint32](0x0011, 0x0101, 0x0001),
		andTC[int64](0x0000001100000011, 0x0000010100000101, 0x0000000100000001),
		andTC[uint64](0x0000001100000011, 0x0000010100000101, 0x0000000100000001),
	}

	for _, test := range tests {
		t.Run(test.String(), test.Run)
	}
}
