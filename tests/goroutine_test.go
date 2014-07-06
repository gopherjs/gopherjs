package tests

import (
	"testing"
	"time"
)

var expectedI int

func checkI(t *testing.T, i int) {
	if i != expectedI {
		t.Errorf("expected %d, got %d", expectedI, i)
	}
	expectedI++
}

func TestDefer(t *testing.T) {
	expectedI = 1
	defer func() {
		checkI(t, 2)
		testDefer1(t)
		checkI(t, 6)
	}()
	checkI(t, 1)
}

func testDefer1(t *testing.T) {
	defer func() {
		checkI(t, 4)
		time.Sleep(0)
		checkI(t, 5)
	}()
	checkI(t, 3)
}

func TestPanic(t *testing.T) {
	expectedI = 1
	defer func() {
		checkI(t, 8)
		err := recover()
		time.Sleep(0)
		checkI(t, err.(int))
	}()
	checkI(t, 1)
	testPanic1(t)
	checkI(t, -1)
}

func testPanic1(t *testing.T) {
	defer func() {
		checkI(t, 6)
		time.Sleep(0)
		err := recover()
		checkI(t, err.(int))
		panic(9)
	}()
	checkI(t, 2)
	testPanic2(t)
	checkI(t, -2)
}

func testPanic2(t *testing.T) {
	defer func() {
		checkI(t, 5)
	}()
	checkI(t, 3)
	time.Sleep(0)
	checkI(t, 4)
	panic(7)
	checkI(t, -3)
}
