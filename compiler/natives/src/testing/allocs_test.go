//go:build js

package testing_test

import "testing"

//gopherjs:replace
func TestAllocsPerRun(t *testing.T) {
	t.Skip("runtime.ReadMemStats() is not supported by GopherJS.")
}
