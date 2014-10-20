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

type SingleValue struct {
	Value uint16
}

type OtherSingleValue struct {
	Value uint16
}

func TestStructKey(t *testing.T) {
	m := make(map[SingleValue]int)
	m[SingleValue{Value: 1}] = 42
	m[SingleValue{Value: 2}] = 43
	if m[SingleValue{Value: 1}] != 42 || m[SingleValue{Value: 2}] != 43 {
		t.Fail()
	}

	m2 := make(map[interface{}]int)
	m2[SingleValue{Value: 1}] = 42
	m2[SingleValue{Value: 2}] = 43
	m2[OtherSingleValue{Value: 1}] = 44
	if m2[SingleValue{Value: 1}] != 42 || m2[SingleValue{Value: 2}] != 43 || m2[OtherSingleValue{Value: 1}] != 44 {
		t.Fail()
	}
}
