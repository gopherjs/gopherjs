package typeparams_test

import (
	"runtime"
	"testing"
)

func _GenericBlocking[T any]() string {
	runtime.Gosched()
	return "Hello, world."
}

// TestBlocking verifies that a generic function correctly resumes after a
// blocking operation.
func TestBlocking(t *testing.T) {
	got := _GenericBlocking[any]()
	want := "Hello, world."
	if got != want {
		t.Fatalf("Got: _GenericBlocking[any]() = %q. Want: %q.", got, want)
	}
}
