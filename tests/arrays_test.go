package tests

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"
)

func TestArrayPointer(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var p1 *[1]int
		if p1 != nil {
			t.Errorf("Zero-value array pointer is not equal to nil: %v", p1)
		}

		var p2 *[1]int = nil
		if p2 != nil {
			t.Errorf("Nil array pointer is not equal to nil: %v", p2)
		}

		p3 := func() *[1]int { return nil }()
		if p3 != nil {
			t.Errorf("Nil array pointer returned from function is not equal to nil: %v", p3)
		}

		if p1 != p3 || p1 != p2 || p2 != p3 {
			t.Errorf("Nil pointers are not equal to each other: %v %v %v", p1, p2, p3)
		}

		if v := reflect.ValueOf(p1); !v.IsNil() {
			t.Errorf("reflect.Value.IsNil() is false for a nil pointer: %v %v", p1, v)
		}

		type arr *[1]int
		var p4 arr = nil

		if v := reflect.ValueOf(p4); !v.IsNil() {
			t.Errorf("reflect.Value.IsNil() is false for a nil pointer: %v %v", p4, v)
		}
	})

	t.Run("pointer-dereference", func(t *testing.T) {
		a1 := [1]int{42}
		aPtr := &a1
		a2 := *aPtr
		if !reflect.DeepEqual(a1, a2) {
			t.Errorf("Array after pointer dereferencing is not equal to the original: %v != %v", a1, a2)
			t.Logf("Pointer: %v", aPtr)
		}
	})

	t.Run("interface-and-back", func(t *testing.T) {
		type arr *[1]int
		tests := []struct {
			name string
			a    arr
		}{{
			name: "not nil",
			a:    &[1]int{42},
		}, {
			name: "nil",
			a:    nil,
		}}
		for _, test := range tests {
			a1 := test.a
			i := interface{}(a1)
			a2 := i.(arr)

			if a1 != a2 {
				t.Errorf("Array pointer is not equal to itself after interface conversion: %v != %v", a1, a2)
				println(a1, a2)
			}
		}
	})

	t.Run("reflect.IsNil", func(t *testing.T) {
	})
}

func TestReflectArraySize(t *testing.T) {
	want := unsafe.Sizeof(int(0)) * 8
	if got := reflect.TypeOf([8]int{}).Size(); got != want {
		t.Errorf("array type size gave %v, want %v", got, want)
	}
}

func TestNilPrototypeNotModifiedByPointer(t *testing.T) {
	const growth = 3

	s1 := []int(nil)
	p1 := &s1
	*p1 = make([]int, 0, growth)
	if c := cap(s1); c != growth {
		t.Errorf(`expected capacity of nil to increase to %d, got %d`, growth, c)
		println("s1:", s1)
	}

	s2 := []int(nil)
	if c := cap(s2); c != 0 {
		t.Errorf(`the capacity of nil must always be zero, it was %d`, c)
		println("s1:", s1)
		println("s2:", s2)
	}
}

func TestNilPrototypeNotModifiedByReflectGrow(t *testing.T) {
	const growth = 3

	s1 := []int(nil)
	v1 := reflect.ValueOf(&s1).Elem()
	v1.Grow(growth)
	if c := cap(s1); c != growth {
		t.Errorf(`expected capacity of nil to increase to %d, got %d`, growth, c)
		println("s1:", s1)
	}

	s2 := []int(nil)
	if c := cap(s2); c != 0 {
		t.Errorf(`the capacity of nil must always be zero, it was %d`, c)
		println("s1:", s1)
		println("s2:", s2)
	}
}

// trySliceArrayCast tests if a slice can be converted to an array
// and either checks if the conversion is successful or if it panics with an expected error.
func trySliceArrayCast[S, A any](t *testing.T, name string, slice S, getArray func(slice S) A, expRuntimeError string) {
	t.Run(name, func(t *testing.T) {
		sv := reflect.ValueOf(slice)
		if sv.Kind() != reflect.Slice {
			t.Fatal(`expected a slice but got`, sv.Kind().String())
		}

		var array A
		r := func() (r any) {
			defer func() { r = recover() }()
			array = getArray(slice)
			return nil
		}()
		if r != nil {
			if len(expRuntimeError) > 0 {
				if rStr := fmt.Sprintf(`%v`, r); rStr != expRuntimeError {
					t.Error(`unexpected panic message:`, r)
					t.Log("\texpected:", expRuntimeError)
				}
			} else {
				t.Error(`unexpected panic:`, r)
			}
			return
		} else if len(expRuntimeError) > 0 {
			t.Error(`did not get expected a panic`)
			t.Log("\texpected:", expRuntimeError)
		}

		av := reflect.ValueOf(array)
		if av.Kind() != reflect.Array {
			t.Fatal(`expected an array but got`, av.Kind().String())
		}

		len := sv.Len()
		if aLen := av.Len(); len > aLen {
			len = aLen
		}

		for i := 0; i < len; i++ {
			if !reflect.DeepEqual(sv.Index(i).Interface(), av.Index(i).Interface()) {
				t.Errorf("slice and array are not equal after conversion (starting at %d):\n"+
					"\tslice: %#v\n\tarray: %#v", i, slice, array)
				return
			}
		}
	})
}

func TestConversionFromSliceToArray(t *testing.T) {
	trySliceArrayCast(t, `4 byte slice to 4 byte array`,
		[]byte{12, 34, 56, 78},
		func(s []byte) [4]byte { return [4]byte(s) }, ``)

	trySliceArrayCast(t, `5 byte slice to 5 byte array`,
		[]byte{12, 34, 56, 78, 90},
		func(s []byte) [5]byte { return [5]byte(s) }, ``)

	trySliceArrayCast(t, `3 byte slice to 3 byte array`,
		[]byte{12, 34, 56},
		func(s []byte) [3]byte { return [3]byte(s) }, ``)

	trySliceArrayCast(t, `5 byte slice to 4 byte array`,
		[]byte{12, 34, 56, 78, 90},
		func(s []byte) [4]byte { return [4]byte(s) }, ``)

	trySliceArrayCast(t, `3 byte slice to 4 byte array`,
		[]byte{12, 34, 56},
		func(s []byte) [4]byte { return [4]byte(s) },
		`runtime error: cannot convert slice with length 3 to array or pointer to array with length 4`)

	trySliceArrayCast(t, `4 byte slice to zero byte array`,
		[]byte{12, 34, 56, 78},
		func(s []byte) [0]byte { return [0]byte(s) }, ``)

	type Cat struct {
		name string
		age  int
	}
	cats := []Cat{
		{name: "Tom", age: 3},
		{name: "Jonesy", age: 5},
		{name: "Sylvester", age: 7},
		{name: "Rita", age: 2},
	}

	trySliceArrayCast(t, `4 Cat slice to 4 Cat array`,
		cats,
		func(s []Cat) [4]Cat { return [4]Cat(s) }, ``)

	trySliceArrayCast(t, `4 *Cat slice to 4 *Cat array`,
		[]*Cat{&cats[0], &cats[1], &cats[2], &cats[3]},
		func(s []*Cat) [4]*Cat { return [4]*Cat(s) }, ``)

	trySliceArrayCast(t, `nil byte slice to zero byte array`,
		[]byte(nil),
		func(s []byte) [0]byte { return [0]byte(s) }, ``)

	trySliceArrayCast(t, `empty byte slice to zero byte array`,
		[]byte{},
		func(s []byte) [0]byte { return [0]byte(s) }, ``)

	trySliceArrayCast(t, `nil byte slice to 4 byte array`,
		[]byte(nil),
		func(s []byte) [4]byte { return [4]byte(s) },
		`runtime error: cannot convert slice with length 0 to array or pointer to array with length 4`)
}
