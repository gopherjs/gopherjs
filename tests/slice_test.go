package tests

import (
	"reflect"
	"testing"
	"unsafe"
)

func Test_SliceClear_Bytes(t *testing.T) {
	var s []byte
	clear(s) // noop

	s = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9}
	clear(s[3:7])
	if want := []byte{1, 2, 3, 0, 0, 0, 0, 8, 9}; !reflect.DeepEqual(s, want) {
		t.Errorf("Got: %v after partial clear, Want: %v", s, want)
	}

	clear(s)
	if want := make([]byte, 9); !reflect.DeepEqual(s, want) {
		t.Errorf("Got: %v after full clear, Want: %v", s, want)
	}
}

func Test_SliceClear_Structs(t *testing.T) {
	type name struct {
		first string
		last  string
	}

	var s []name
	clear(s) // noop

	s = []name{
		{first: `bob`, last: `bobson`},
		{first: `jill`, last: `jillton`},
		{first: `brian`, last: `o'brian`},
		{first: `brian`, last: `o'brian`},
	}
	clear(s[1:3])
	if want := []name{s[0], {}, {}, s[3]}; !reflect.DeepEqual(s, want) {
		t.Errorf("Got: %v after partial clear, Want: %v", s, want)
	}

	clear(s)
	if want := make([]name, 4); !reflect.DeepEqual(s, want) {
		t.Errorf("Got: %v after full clear, Want: %v", s, want)
	}
}

func Test_UnsafeSlice(t *testing.T) {
	var a [10]byte
	x := a[1:4]
	p := &x[2]
	s := unsafe.Slice(p, 6) // s == a[3:9]

	for i := range s {
		s[i] = byte(i + 10)
	}

	if want := [10]byte{0, 0, 0, 10, 11, 12, 13, 14, 15, 0}; !reflect.DeepEqual(a, want) {
		t.Errorf("Got: %v after unsafe slice, Want: %v", a, want)
	}
}

func Test_OffsetSliceToArray(t *testing.T) {
	s := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	s2 := s[2:6]
	if want := []byte{3, 4, 5, 6}; !reflect.DeepEqual(s2, want) {
		t.Errorf("Got: %v after getting sub-slice, Want: %v", s2, want)
	}

	// Originally there was an issue where this cast would clone the underlying array
	// from s2 and copy it without the offset so the result would be `[4]byte{1, 2, 3, 4}`
	// instead of the correct offset values.
	a := [4]byte(s2)
	if want := [4]byte{3, 4, 5, 6}; !reflect.DeepEqual(a, want) {
		t.Errorf("Got: %v after casting to array, Want: %v", a, want)
	}

	s2[1] = 42
	if want, got := byte(4), a[1]; want != got {
		t.Errorf("Got: %v after casting to array, Want: %v", got, want)
	}
}
