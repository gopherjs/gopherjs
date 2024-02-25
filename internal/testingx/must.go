// Package testingx provides helpers for use with the testing package.
package testingx

import "testing"

// Must provides a concise way to handle handle returned error in tests that
// "should never happen"©.
//
// This function can be used in test case setup that can be presumed to be
// correct, but technically may return an error. This function MUST NOT be used
// to check for test case conditions themselves because it provides a generic,
// nondescript test error message.
//
//	func startServer(addr string) (*server, err)
//	mustServer := testingx.Must[*server](t)
//	mustServer(startServer(":8080"))
func Must[T any](t *testing.T) func(v T, err error) T {
	return func(v T, err error) T {
		if err != nil {
			t.Fatalf("Got: unexpected error: %s. Want: no error.", err)
		}
		return v
	}
}
