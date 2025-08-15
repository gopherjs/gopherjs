//go:build js

package time_test

import "testing"

func TestZeroTimer(t *testing.T) {
	t.Skip(`This test is very slow (about 19 mins)`)
}

//gopherjs:replace
func TestSleep(t *testing.T) {
	t.Skip("time.Now() is not accurate enough for the test")
}
