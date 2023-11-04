package typeparams_test

import (
	"fmt"
	"testing"
)

func TestMake(t *testing.T) {
	t.Run("slice", func(t *testing.T) {
		tests := []struct {
			slice   []int
			wantStr string
			wantLen int
			wantCap int
		}{{
			slice:   make([]int, 1),
			wantStr: "[]int{0}",
			wantLen: 1,
			wantCap: 1,
		}, {
			slice:   make([]int, 1, 2),
			wantStr: "[]int{0}",
			wantLen: 1,
			wantCap: 2,
		}}

		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				if got := fmt.Sprintf("%#v", test.slice); got != test.wantStr {
					t.Errorf("Got: fmt.Sprint(%v) = %q. Want: %q.", test.slice, got, test.wantStr)
				}
				if got := len(test.slice); got != test.wantLen {
					t.Errorf("Got: len(%v) = %d. Want: %d.", test.slice, got, test.wantLen)
				}
				if got := cap(test.slice); got != test.wantCap {
					t.Errorf("Got: cap(%v) = %d. Want: %d.", test.slice, got, test.wantCap)
				}
			})
		}
	})

	t.Run("map", func(t *testing.T) {
		tests := []map[int]int{
			make(map[int]int),
			make(map[int]int, 1),
		}

		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				want := "map[int]int{}"
				got := fmt.Sprintf("%#v", test)
				if want != got {
					t.Errorf("Got: fmt.Sprint(%v) = %q. Want: %q.", test, got, want)
				}
			})
		}
	})

	t.Run("chan", func(t *testing.T) {
		tests := []struct {
			ch      chan int
			wantCap int
		}{{
			ch:      make(chan int),
			wantCap: 0,
		}, {
			ch:      make(chan int, 1),
			wantCap: 1,
		}}

		for i, test := range tests {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				wantStr := "chan int"
				if got := fmt.Sprintf("%T", test.ch); got != wantStr {
					t.Errorf("Got: fmt.Sprint(%v) = %q. Want: %q.", test.ch, got, wantStr)
				}
				if got := cap(test.ch); got != test.wantCap {
					t.Errorf("Got: cap(%v) = %d. Want: %d.", test.ch, got, test.wantCap)
				}
			})
		}
	})
}
