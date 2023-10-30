package typeparams_test

import (
	"fmt"
	"math"
	"reflect"
	"runtime"
	"testing"

	"github.com/gopherjs/gopherjs/js"
)

// checkConversion is a general type conversion result checker.
//
// The expectation is that got and want have the same underlying Go type, and
// they contain equal values. Src is the original value before type conversion,
// provided for error message purposes.
//
// Note that this function is suitable for checking most conversion results
// except conversion to interfaces. This is because use of reflect APIs requires
// conversion to `any` interface, which must be assumed correct for this test to
// be meaningful.
func checkConversion(t *testing.T, src, got, want any) {
	t.Helper()
	if reflect.TypeOf(got) != reflect.TypeOf(want) {
		t.Errorf("Got: %v. Want: converted type is: %v.", reflect.TypeOf(got), reflect.TypeOf(want))
	}

	if !reflect.DeepEqual(want, got) {
		t.Errorf("Got: %[1]T(%#[1]v). Want: %[2]T(%#[2]v) convert to %[3]T(%#[3]v).", got, src, want)
	}
}

// conversionTest is a common interface for type conversion test cases.
type conversionTest interface {
	Run(t *testing.T)
}

type numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64 |
		~uintptr
}

type numericConversion[srcType numeric, dstType numeric] struct {
	src   srcType
	want  dstType
	quirk bool
}

func (tc numericConversion[srcType, dstType]) Run(t *testing.T) {
	if tc.quirk && runtime.Compiler != "gopherjs" {
		t.Skip("GopherJS-only test")
	}

	checkConversion(t, tc.src, dstType(tc.src), tc.want)
}

type complex interface {
	~complex64 | ~complex128
}

type complexConversion[srcType complex, dstType complex] struct {
	src  srcType
	want dstType
}

func (tc complexConversion[srcType, dstType]) Run(t *testing.T) {
	checkConversion(t, tc.src, dstType(tc.src), tc.want)
}

type stringLike interface {
	// Ideally, we would test conversions from all integer types. unfortunately,
	// that trips up the stringintconv check in `go vet` that is ran by `go test`
	// by default. Unfortunately, there is no way to selectively suppress that
	// check.
	// ~int | ~int8 | ~int16 | ~int32 | ~int64 |
	// ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
	// ~uintptr |
	byte | rune |
		~[]byte | ~[]rune | ~string
}

type stringConversion[srcType stringLike, dstType ~string] struct {
	src  srcType
	want dstType
}

func (tc stringConversion[srcType, dstType]) Run(t *testing.T) {
	checkConversion(t, tc.src, dstType(tc.src), tc.want)
}

type boolConversion[srcType ~bool, dstType ~bool] struct {
	src  srcType
	want dstType
}

func (tc boolConversion[srcType, dstType]) Run(t *testing.T) {
	checkConversion(t, tc.src, dstType(tc.src), tc.want)
}

// asString returns a string that reflects internal representation of the object.
//
// There is not specific guarantees about the string format, expect that if two
// strings match, the two objects _almost certainly_ are deeply equal.
func asString(o *js.Object) string {
	f := js.Global.Get("Function").New("o", `
	const seen = [];
	// JSON.stringify can't deal with circular references, which GopherJS objects
	// can have. So when the same object is seen more than once we replace it with
	// a string stub.
	const suppressCycles = (key, value) => {
		if (typeof value !== 'object') {
			return value;
		}
		const idx = seen.indexOf(value);
		if (idx !== -1) {
			return "[Cycle " + idx + "]"
		}
		seen.push(value);
		return value;
	}
	return JSON.stringify(o, suppressCycles);
	`)
	return f.Invoke(o).String()
}

type interfaceConversion[srcType any] struct {
	src srcType
}

func (tc interfaceConversion[srcType]) Run(t *testing.T) {
	// All of the following expressions are semantically equivalent, but may be
	// compiled by GopherJS differently, so we test all of them.
	var got1 any
	got1 = tc.src
	var got2 any = tc.src
	var got3 any = any(tc.src)

	for i, got := range []any{got1, got2, got3} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			checkConversion(t, tc.src, got, tc.src)

			concrete := got.(srcType) // Must not panic.
			if runtime.Compiler == "gopherjs" {
				// Can't use  reflect.DeepEqual() here because it itself relies on
				// conversion to interface, so instead we do some JS-level introspection.
				srcRepr := asString(js.InternalObject(tc.src))
				concreteRepr := asString(js.InternalObject(concrete))
				if srcRepr == "" {
					t.Fatalf("Got: internal representation of the original value is empty. Want: not empty.")
				}
				if concreteRepr != srcRepr {
					t.Errorf("Got: result of type assertion %q is not equal to the original value %q. Want: values are equal.", concreteRepr, srcRepr)
				}
			}
		})
	}
}

func TestConversion(t *testing.T) {
	type i64 int64
	type i32 int32
	type f64 float64
	type f32 float32
	type c64 complex64
	type c128 complex128
	type str string
	type b bool
	type st struct {
		s string
		i int
	}

	tests := []conversionTest{
		// $convertToInt64
		numericConversion[int, int64]{src: 0x7FFFFFFF, want: 0x7FFFFFFF},
		numericConversion[int64, uint64]{src: -0x8000000000000000, want: 0x8000000000000000},
		numericConversion[uint, int64]{src: 0xFFFFFFFF, want: 0xFFFFFFFF},
		numericConversion[uint64, int64]{src: 0xFFFFFFFFFFFFFFFF, want: -1},
		numericConversion[uint64, uint64]{src: 0xFFFFFFFFFFFFFFFF, want: 0xFFFFFFFFFFFFFFFF},
		numericConversion[uintptr, uint64]{src: 0xFFFFFFFF, want: 0xFFFFFFFF},
		numericConversion[uintptr, uint64]{src: reflect.ValueOf(&struct{}{}).Pointer(), want: 0x1, quirk: true},
		numericConversion[float32, int64]{src: 2e10, want: 20000000000},
		numericConversion[float64, int64]{src: 2e10, want: 20000000000},
		numericConversion[int64, i64]{src: 1, want: 1},
		numericConversion[i64, int64]{src: 1, want: 1},
		// $convertToNativeInt
		numericConversion[int64, int32]{src: math.MaxInt64, want: -1},
		numericConversion[int64, int32]{src: -100, want: -100},
		numericConversion[int64, int32]{src: 0x00C0FFEE4B1D4B1D, want: 0x4B1D4B1D},
		numericConversion[int32, int16]{src: 0x0BAD4B1D, want: 0x4B1D},
		numericConversion[int16, int8]{src: 0x4B1D, want: 0x1D},
		numericConversion[uint64, uint32]{src: 0xDEADC0DE00C0FFEE, want: 0x00C0FFEE},
		numericConversion[uint32, uint16]{src: 0xDEADC0DE, want: 0xC0DE},
		numericConversion[uint16, uint8]{src: 0xC0DE, want: 0xDE},
		numericConversion[float32, int32]{src: 12345678.12345678, want: 12345678},
		numericConversion[float32, int16]{src: 12345678.12345678, want: 24910},
		numericConversion[float64, int32]{src: 12345678.12345678, want: 12345678},
		numericConversion[float64, int16]{src: 12345678.12345678, want: 24910},
		numericConversion[int32, int]{src: 0x00C0FFEE, want: 0x00C0FFEE},
		numericConversion[uint32, uint]{src: 0x00C0FFEE, want: 0x00C0FFEE},
		numericConversion[uint32, uintptr]{src: 0x00C0FFEE, want: 0x00C0FFEE},
		numericConversion[int32, i32]{src: 0x00C0FFEE, want: 0x00C0FFEE},
		numericConversion[i32, int32]{src: 0x00C0FFEE, want: 0x00C0FFEE},
		numericConversion[uint32, int32]{src: 0xFFFFFFFF, want: -1},
		numericConversion[uint16, int16]{src: 0xFFFF, want: -1},
		numericConversion[uint8, int8]{src: 0xFF, want: -1},
		// $convertToFloat
		numericConversion[float64, float32]{src: 12345678.1234567890, want: 12345678.0},
		numericConversion[int64, float32]{src: 123456789, want: 123456792.0},
		numericConversion[int32, float32]{src: 12345678, want: 12345678.0},
		numericConversion[f32, float32]{src: 12345678.0, want: 12345678.0},
		numericConversion[float32, f32]{src: 12345678.0, want: 12345678.0},
		numericConversion[float32, float64]{src: 1234567.125000, want: 1234567.125000},
		numericConversion[int64, float64]{src: 12345678, want: 12345678.0},
		numericConversion[int32, float64]{src: 12345678, want: 12345678.0},
		numericConversion[f64, float64]{src: 12345678.0, want: 12345678.0},
		numericConversion[float64, f64]{src: 12345678.0, want: 12345678.0},
		// $convertToComplex
		complexConversion[complex64, complex128]{src: 1 + 1i, want: 1 + 1i},
		complexConversion[complex128, complex64]{src: 1 + 1i, want: 1 + 1i},
		complexConversion[complex128, c128]{src: 1 + 1i, want: 1 + 1i},
		complexConversion[complex64, c64]{src: 1 + 1i, want: 1 + 1i},
		// $convertToString
		stringConversion[str, string]{src: "abc", want: "abc"},
		stringConversion[string, str]{src: "abc", want: "abc"},
		stringConversion[rune, string]{src: 'a', want: "a"},
		stringConversion[byte, string]{src: 'a', want: "a"},
		stringConversion[[]byte, string]{src: []byte{'a', 'b', 'c'}, want: "abc"},
		stringConversion[[]rune, string]{src: []rune{'a', 'b', 'c'}, want: "abc"},
		// $convertToBool
		boolConversion[b, bool]{src: true, want: true},
		boolConversion[b, bool]{src: false, want: false},
		boolConversion[bool, b]{src: true, want: true},
		boolConversion[bool, b]{src: false, want: false},
		// $convertToInterface
		interfaceConversion[int]{src: 1},
		interfaceConversion[string]{src: "abc"},
		interfaceConversion[string]{src: "abc"},
		interfaceConversion[st]{src: st{s: "abc", i: 1}},
		interfaceConversion[error]{src: fmt.Errorf("test error")},
		interfaceConversion[*js.Object]{src: js.Global},
		interfaceConversion[*int]{src: func(i int) *int { return &i }(1)},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%T", test), test.Run)
	}
}
