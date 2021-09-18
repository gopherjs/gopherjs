//go:build js
// +build js

package time_test

import (
	"testing"
)

func TestSleep(t *testing.T) {
	t.Skip("time.Now() is not accurate enough for the test")
}

func TestEnvTZUsage(t *testing.T) {
	t.Skip("TZ environment variable in not applicable in the browser context.")
}
