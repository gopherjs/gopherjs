package tests

import (
	"reflect"
	"testing"
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
