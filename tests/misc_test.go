package tests

import "testing"

func TestSyntax1(t *testing.T) {
	a := 42
	if *&*&a != 42 {
		t.Fail()
	}
}

func TestPointerEquality(t *testing.T) {
	a := [3]int{1, 2, 3}
	if &a[0] != &a[0] || &a[:][0] != &a[0] || &a[:][0] != &a[:][0] {
		t.Fail()
	}
}
