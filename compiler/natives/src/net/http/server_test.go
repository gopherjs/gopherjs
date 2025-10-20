//go:build js

package http_test

import "testing"

func TestTimeoutHandlerSuperfluousLogs(t *testing.T) {
	// The test expects nested anonymous functions to be named "Foo.func1.2",
	// bug GopherJS generates "Foo.func1.func2". Otherwise the test works as
	// expected.
	t.Skip("GopherJS uses different synthetic function names.")
}

func TestHTTP2WriteDeadlineExtendedOnNewRequest(t *testing.T) {
	// Test depends on httptest.NewUnstartedServer
	t.Skip("Network access not supported by GopherJS.")
}
