//go:build js

package time_test

import "testing"

//gopherjs:replace
func TestSleep(t *testing.T) {
	t.Skip("time.Now() is not accurate enough for the test")
}

//gopherjs:replace
func TestEnvTZUsage(t *testing.T) {
	t.Skip("TZ environment variable in not applicable in the browser context.")
}
