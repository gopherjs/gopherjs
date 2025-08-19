//go:build js

package http_test

import "testing"

func TestTimeoutHandlerSuperfluousLogs(t *testing.T) {
	t.Skip("https://github.com/gopherjs/gopherjs/issues/1085")
}

func TestHTTP2WriteDeadlineExtendedOnNewRequest(t *testing.T) {
	// Test depends on httptest.NewUnstartedServer
	t.Skip("Network access not supported by GopherJS.")
}
