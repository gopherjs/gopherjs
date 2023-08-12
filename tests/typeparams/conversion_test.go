package typeparams_test

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
	"testing"
)

type numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 |
		~uintptr
}

type converter interface {
	Src() any
	Got() any
	Want() any
	Quirk() bool // Tests GopherJS-specific behavior.
}

type numericConverter[srcType numeric, dstType numeric] struct {
	src   srcType
	want  dstType
	quirk bool
}

func (tc numericConverter[srcType, dstType]) Src() any {
	return tc.src
}

func (tc numericConverter[srcType, dstType]) Got() any {
	return dstType(tc.src)
}

func (tc numericConverter[srcType, dstType]) Want() any {
	return tc.want
}

func (tc numericConverter[srcType, dstType]) Quirk() bool {
	return tc.quirk
}

func TestConversion(t *testing.T) {
	type i64 int64
	type i32 int32
	tests := []converter{
		// $convertToInt64
		numericConverter[int, int64]{src: 0x7FFFFFFF, want: 0x7FFFFFFF},
		numericConverter[int64, uint64]{src: -0x8000000000000000, want: 0x8000000000000000},
		numericConverter[uint, int64]{src: 0xFFFFFFFF, want: 0xFFFFFFFF},
		numericConverter[uint64, int64]{src: 0xFFFFFFFFFFFFFFFF, want: -1},
		numericConverter[uint64, uint64]{src: 0xFFFFFFFFFFFFFFFF, want: 0xFFFFFFFFFFFFFFFF},
		numericConverter[uintptr, uint64]{src: 0xFFFFFFFF, want: 0xFFFFFFFF},
		numericConverter[uintptr, uint64]{src: reflect.ValueOf(&struct{}{}).Pointer(), want: 0x1, quirk: true},
		numericConverter[float32, int64]{src: 2e10, want: 20000000000},
		numericConverter[float64, int64]{src: 2e10, want: 20000000000},
		numericConverter[int64, i64]{src: 1, want: 1},
		numericConverter[i64, int64]{src: 1, want: 1},
		// $convertToNativeInt
		numericConverter[int64, int32]{src: math.MaxInt64, want: -1},
		numericConverter[int64, int32]{src: -100, want: -100},
		numericConverter[int64, int32]{src: 0x00C0FFEE4B1D4B1D, want: 0x4B1D4B1D},
		numericConverter[int32, int16]{src: 0x0BAD4B1D, want: 0x4B1D},
		numericConverter[int16, int8]{src: 0x4B1D, want: 0x1D},
		numericConverter[uint64, uint32]{src: 0xDEADC0DE00C0FFEE, want: 0x00C0FFEE},
		numericConverter[uint32, uint16]{src: 0xDEADC0DE, want: 0xC0DE},
		numericConverter[uint16, uint8]{src: 0xC0DE, want: 0xDE},
		numericConverter[float32, int32]{src: 12345678.12345678, want: 12345678},
		numericConverter[float32, int16]{src: 12345678.12345678, want: 24910},
		numericConverter[float64, int32]{src: 12345678.12345678, want: 12345678},
		numericConverter[float64, int16]{src: 12345678.12345678, want: 24910},
		numericConverter[int32, int]{src: 0x00C0FFEE, want: 0x00C0FFEE},
		numericConverter[uint32, uint]{src: 0x00C0FFEE, want: 0x00C0FFEE},
		numericConverter[uint32, uintptr]{src: 0x00C0FFEE, want: 0x00C0FFEE},
		numericConverter[int32, i32]{src: 0x00C0FFEE, want: 0x00C0FFEE},
		numericConverter[i32, int32]{src: 0x00C0FFEE, want: 0x00C0FFEE},
		numericConverter[uint32, int32]{src: 0xFFFFFFFF, want: -1},
		numericConverter[uint16, int16]{src: 0xFFFF, want: -1},
		numericConverter[uint8, int8]{src: 0xFF, want: -1},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%T", test), func(t *testing.T) {
			if test.Quirk() && runtime.Compiler != "gopherjs" {
				t.Skip("GopherJS-only test")
			}
			got := test.Got()
			want := test.Want()
			if !reflect.DeepEqual(want, got) {
				t.Errorf("Want: %[1]T(%#[1]v) convert to %[2]T(%#[2]v). Got: %[3]T(%#[3]v)", test.Src(), want, got)
			}
		})
	}
}
