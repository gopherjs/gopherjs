package tests

import (
	"reflect"
	"testing"
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
