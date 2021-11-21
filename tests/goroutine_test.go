package tests

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/gopherjs/gopherjs/js"
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

func TestPanicAdvanced(t *testing.T) {
	expectedI = 1
	defer func() {
		recover()
		checkI(t, 3)
		testPanicAdvanced2(t)
		checkI(t, 6)
	}()
	testPanicAdvanced1(t)
	checkI(t, -1)
}

func testPanicAdvanced1(t *testing.T) {
	defer func() {
		checkI(t, 2)
	}()
	checkI(t, 1)
	panic("")
}

func testPanicAdvanced2(t *testing.T) {
	defer func() {
		checkI(t, 5)
	}()
	checkI(t, 4)
}

func TestPanicIssue1030(t *testing.T) {
	throwException := func() {
		t.Log("Will throw now...")
		js.Global.Call("eval", "throw 'original panic';")
	}

	wrapException := func() {
		defer func() {
			err := recover()
			if err == nil {
				t.Fatal("Should never happen: no original panic.")
			}
			t.Log("Got original panic: ", err)
			panic("replacement panic")
		}()

		throwException()
	}

	panicing := false

	expectPanic := func() {
		defer func() {
			t.Log("No longer panicing.")
			panicing = false
		}()
		defer func() {
			err := recover()
			if err == nil {
				t.Fatal("Should never happen: no wrapped panic.")
			}
			t.Log("Got wrapped panic: ", err)
		}()

		wrapException()
	}

	expectPanic()

	if panicing {
		t.Fatal("Deferrals were not executed correctly!")
	}
}

func TestSelect(t *testing.T) {
	expectedI = 1
	a := make(chan int)
	b := make(chan int)
	c := make(chan int)
	go func() {
		select {
		case <-a:
		case <-b:
		}
	}()
	go func() {
		checkI(t, 1)
		a <- 1
		select {
		case b <- 1:
			checkI(t, -1)
		default:
			checkI(t, 2)
		}
		c <- 1
	}()
	<-c
	checkI(t, 3)
}

func TestCloseAfterReceiving(t *testing.T) {
	ch := make(chan struct{})
	go func() {
		<-ch
		close(ch)
	}()
	ch <- struct{}{}
}

func TestDeferWithBlocking(t *testing.T) {
	ch := make(chan struct{})
	go func() { ch <- struct{}{} }()
	defer func() { <-ch }()
	fmt.Print("")
	return
}

// counter, sideEffect and withBlockingDeferral are defined as top-level symbols
// to make compiler generate simplest code possible without any closures.
var counter = 0

func sideEffect() int {
	counter++
	return 42
}

func withBlockingDeferral() int {
	defer time.Sleep(0)
	return sideEffect()
}

func TestReturnWithBlockingDefer(t *testing.T) {
	// See: https://github.com/gopherjs/gopherjs/issues/603.
	counter = 0

	got := withBlockingDeferral()
	if got != 42 {
		t.Errorf("Unexpected return value %v. Want: 42.", got)
	}
	if counter != 1 {
		t.Errorf("Return value was computed %d times. Want: exactly 1.", counter)
	}
}

func BenchmarkGoroutineSwitching(b *testing.B) {
	// This benchmark is designed to measure the cost of goroutine switching.
	// The two goroutines communicate through an unbuffered channel, which forces
	// the control to be passed between them on each iteraction of the benchmark.
	// Although the cost of channel operations is also included in the measurement,
	// it still allows relative comparison of changes to goroutine scheduling
	// performance.
	c := make(chan bool)
	go func() {
		for i := 0; i < b.N; i++ {
			c <- true
		}
		close(c)
	}()

	b.ResetTimer()
	count := 0
	for range c {
		count++
	}
}

func TestEventLoopStarvation(t *testing.T) {
	// See: https://github.com/gopherjs/gopherjs/issues/1078.
	c := make(chan bool)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	go func() {
		for {
			select {
			case c <- true:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		for {
			select {
			case <-c:
			case <-ctx.Done():
				return
			}
		}
	}()
	<-ctx.Done()
}

func TestGoroutineBuiltin(t *testing.T) {
	// Test that a built-in function can be a goroutine body.
	// https://github.com/gopherjs/gopherjs/issues/547.
	c := make(chan bool)
	go close(c)
	<-c // Wait until goroutine executes successfully.
}

func TestGoroutineJsObject(t *testing.T) {
	// Test that js.Object methods can be a goroutine body.
	// https://github.com/gopherjs/gopherjs/issues/547.
	if !(runtime.GOOS == "js" || runtime.GOARCH == "js") {
		t.Skip("Test requires GopherJS")
	}
	o := js.Global.Get("Object").New()
	go o.Set("x", "y")
	// Wait until the goroutine executes successfully. Can't use locks here
	// because goroutine body must be a bare js.Object method call.
	for o.Get("x").String() != "y" {
		runtime.Gosched()
	}
}
