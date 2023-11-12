package typeparams_test

import (
	"fmt"
	"reflect"
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

func _len[T []int | *[3]int | map[int]int | chan int | string](x T) int {
	return len(x)
}

func TestLen(t *testing.T) {
	ch := make(chan int, 2)
	ch <- 1

	tests := []struct {
		desc string
		got  int
		want int
	}{{
		desc: "string",
		got:  _len("abcd"),
		want: 4,
	}, {
		desc: "[]int",
		got:  _len([]int{1, 2, 3}),
		want: 3,
	}, {
		desc: "[3]int",
		got:  _len(&[3]int{1}),
		want: 3,
	}, {
		desc: "map[int]int",
		got:  _len(map[int]int{1: 1, 2: 2}),
		want: 2,
	}, {
		desc: "chan int",
		got:  _len(ch),
		want: 1,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("Got: len() = %d. Want: %d.", test.got, test.want)
			}
		})
	}
}

func _cap[T []int | *[3]int | [3]int | chan int](x T) int {
	return cap(x)
}

func TestCap(t *testing.T) {
	ch := make(chan int, 2)
	ch <- 1

	tests := []struct {
		desc string
		got  int
		want int
	}{{
		desc: "[]int",
		got:  _cap([]int{1, 2, 3}),
		want: 3,
	}, {
		desc: "*[3]int",
		got:  _cap(&[3]int{1}),
		want: 3,
	}, {
		desc: "[3]int",
		got:  _cap([3]int{1}),
		want: 3,
	}, {
		desc: "chan int",
		got:  _cap(ch),
		want: 2,
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("Got: len() = %d. Want: %d.", test.got, test.want)
			}
		})
	}
}

func _new[T any]() *T {
	return new(T)
}

func TestNew(t *testing.T) {
	type S struct{ i int }

	tests := []struct {
		desc string
		got  any
		want any
	}{{
		desc: "struct S",
		got:  *_new[S](),
		want: S{},
	}, {
		desc: "[3]int",
		got:  *_new[[3]int](),
		want: [3]int{},
	}, {
		desc: "int",
		got:  *_new[int](),
		want: int(0),
	}}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if !reflect.DeepEqual(test.got, test.want) {
				t.Errorf("Got: new(%T) = %#v. Want: %#v.", test.want, test.got, test.want)
			}
		})
	}
}
