//go:build js

package time_test

import "testing"

//gopherjs:replace
func TestSleep(t *testing.T) {
	t.Skip("time.Now() is not accurate enough for the test")
}

// gopherjs:replace
func TestZeroTimer(t *testing.T) {
	t.Skip(`This test is very slow (about 19 mins)`)
}
