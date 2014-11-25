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
