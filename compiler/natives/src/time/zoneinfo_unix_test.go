//go:build js && unix && !ios && !android

package time_test

import "testing"

func TestEnvTZUsage(t *testing.T) {
	t.Skip("TZ environment variable in not applicable in the browser context.")
}
