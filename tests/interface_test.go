package tests

import (
	"fmt"
	"testing"
)

type Struct struct {
	Name string
}

func (s Struct) SetName(n string) {
	s.Name = n
}

type SetName interface {
	SetName(n string)
}

func TestAssignStructValInterface(t *testing.T) {
	s := Struct{
		Name: "Rob",
	}

	var i1 interface{} = s
	var i2 interface{} = i1

	s.Name = "Pike"

	ss := fmt.Sprintf("%#v", s)
	i1s := fmt.Sprintf("%#v", i1)
	i2s := fmt.Sprintf("%#v", i2)

	if exp := "tests.Struct{Name:\"Pike\"}"; ss != exp {
		t.Fatalf("ss should have been %q; got %q", exp, ss)
	}

	iexp := "tests.Struct{Name:\"Rob\"}"

	if i1s != iexp {
		t.Fatalf("is should have been %q; got %q", iexp, i1s)
	}

	if i2s != iexp {
		t.Fatalf("is should have been %q; got %q", iexp, i2s)
	}
}

func TestStructValInterfaceMethodCall(t *testing.T) {
	var i SetName = Struct{
		Name: "Rob",
	}

	i.SetName("Pike")

	is := fmt.Sprintf("%#v", i)

	if exp := "tests.Struct{Name:\"Rob\"}"; is != exp {
		t.Fatalf("is should have been %q; got %q", exp, is)
	}
}

func TestAssignArrayInterface(t *testing.T) {
	a := [2]int{1, 2}

	var i1 interface{} = a
	var i2 interface{} = i1

	a[0] = 0

	as := fmt.Sprintf("%#v", a)
	i1s := fmt.Sprintf("%#v", i1)
	i2s := fmt.Sprintf("%#v", i2)

	if exp := "[2]int{0, 2}"; as != exp {
		t.Fatalf("ss should have been %q; got %q", exp, as)
	}

	iexp := "[2]int{1, 2}"

	if i1s != iexp {
		t.Fatalf("is should have been %q; got %q", iexp, i1s)
	}

	if i2s != iexp {
		t.Fatalf("is should have been %q; got %q", iexp, i2s)
	}
}
