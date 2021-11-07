package tests

import (
	"testing"
	"time"
)

func inner(ch chan struct{}, b bool) ([]byte, error) {
	// ensure gopherjs thinks that this inner function can block
	if b {
		<-ch
	}
	return []byte{}, nil
}

// this function's call to inner never blocks, but the deferred
// statement does.
func outer(ch chan struct{}, b bool) ([]byte, error) {
	defer func() {
		<-ch
	}()

	return inner(ch, b)
}

func TestBlockingInDefer(t *testing.T) {
	defer func() {
		if x := recover(); x != nil {
			t.Errorf("run time panic: %v", x)
		}
	}()

	ch := make(chan struct{})
	b := false

	go func() {
		time.Sleep(5 * time.Millisecond)
		ch <- struct{}{}
	}()

	outer(ch, b)
}

func TestIssue1083(t *testing.T) {
	// https://github.com/gopherjs/gopherjs/issues/1083
	var block = make(chan bool)

	recoverCompleted := false

	recoverAndBlock := func() {
		defer func() {}()
		recover()
		block <- true
		recoverCompleted = true
	}

	handle := func() {
		defer func() {}()
		panic("expected panic")
	}

	serve := func() {
		defer recoverAndBlock()
		handle()
		t.Fatal("This line must never execute.")
	}

	go func() { <-block }()

	serve()
	if !recoverCompleted {
		t.Fatal("Recovery function did not execute fully.")
	}
}
