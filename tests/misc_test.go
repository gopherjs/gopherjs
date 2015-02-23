package tests

import (
	"math"
	"strings"
	"testing"
)

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

func TestSelectOnNilChan(t *testing.T) {
	var c1 chan bool
	c2 := make(chan bool)

	go func() {
		close(c2)
	}()

	select {
	case <-c1:
		t.Fail()
	case <-c2:
		// ok
	}
}

type StructA struct {
	x int
}

type StructB struct {
	StructA
}

func TestEmbeddedStruct(t *testing.T) {
	a := StructA{
		42,
	}
	b := StructB{
		StructA: a,
	}
	b.x = 0
	if a.x != 42 {
		t.Fail()
	}
}

func TestMapStruct(t *testing.T) {
	a := StructA{
		42,
	}
	m := map[int]StructA{
		1: a,
	}
	m[2] = a
	a.x = 0
	if m[1].x != 42 || m[2].x != 42 {
		t.Fail()
	}
}

func TestUnnamedParameters(t *testing.T) {
	ok := false
	defer func() {
		if !ok {
			t.Fail()
		}
	}()
	blockingWithUnnamedParameter(false) // used to cause non-blocking call error, which is ignored by testing
	ok = true
}

func blockingWithUnnamedParameter(bool) {
	c := make(chan int, 1)
	c <- 42
}

func TestGotoLoop(t *testing.T) {
	goto loop
loop:
	for i := 42; ; {
		if i != 42 {
			t.Fail()
		}
		break
	}
}

func TestMaxUint64(t *testing.T) {
	if math.MaxUint64 != 18446744073709551615 {
		t.Fail()
	}
}

func TestCopyBuiltin(t *testing.T) {
	{
		s := []string{"a", "b", "c"}
		copy(s, s[1:])
		if s[0] != "b" || s[1] != "c" || s[2] != "c" {
			t.Fail()
		}
	}
	{
		s := []string{"a", "b", "c"}
		copy(s[1:], s)
		if s[0] != "a" || s[1] != "a" || s[2] != "b" {
			t.Fail()
		}
	}
}

func TestPointerOfStructConversion(t *testing.T) {
	type A struct {
		Value int
	}

	type B A

	a1 := &A{Value: 1}
	b1 := (*B)(a1)
	b1.Value = 2
	a2 := (*A)(b1)
	a2.Value = 3
	b2 := (*B)(a2)
	b2.Value = 4
	if a1 != a2 || b1 != b2 || a1.Value != 4 || a2.Value != 4 || b1.Value != 4 || b2.Value != 4 {
		t.Fail()
	}
}

func TestCompareStruct(t *testing.T) {
	type A struct {
		Value int
	}

	a := A{42}
	var b interface{} = a
	x := A{0}

	if a != b || a == x || b == x {
		t.Fail()
	}
}

func TestLoopClosure(t *testing.T) {
	type S struct{ fn func() int }
	var fns []*S
	for i := 0; i < 2; i++ {
		z := i
		fns = append(fns, &S{
			fn: func() int {
				return z
			},
		})
	}
	for i, f := range fns {
		if f.fn() != i {
			t.Fail()
		}
	}
}

func TestNilInterfaceError(t *testing.T) {
	defer func() {
		if err := recover(); err == nil || !strings.Contains(err.(error).Error(), "nil pointer dereference") {
			t.Fail()
		}
	}()
	var err error
	err.Error()
}

func TestNilAtLhs(t *testing.T) {
	type F func(string) string
	var f F
	if nil != f {
		t.Fail()
	}
}
