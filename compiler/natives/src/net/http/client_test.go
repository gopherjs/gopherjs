//go:build js

package http_test

import "testing"

func testClientTimeout(t *testing.T, h2 bool) {
	// The original test expects Client.Timeout error to be returned, but under
	// GopherJS an "i/o timeout" error is frequently returned. Otherwise the test
	// seems to be working correctly.
	t.Skip("Flaky test under GopherJS.")
}

func testClientTimeout_Headers(t *testing.T, h2 bool) {
	// The original test expects Client.Timeout error to be returned, but under
	// GopherJS an "i/o timeout" error is frequently returned. Otherwise the test
	// seems to be working correctly.
	t.Skip("Flaky test under GopherJS.")
}
