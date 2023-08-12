package typeparams_test

import (
	"fmt"
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
