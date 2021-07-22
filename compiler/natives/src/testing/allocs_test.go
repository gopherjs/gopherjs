//go:build js
// +build js

package testing_test

import "testing"

func TestAllocsPerRun(t *testing.T) {
	t.Skip("runtime.ReadMemStats() is not supported by GopherJS.")
}
