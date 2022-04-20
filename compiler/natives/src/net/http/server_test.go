//go:build js
// +build js

package http_test

import "testing"

func TestTimeoutHandlerSuperfluousLogs(t *testing.T) {
	t.Skip("https://github.com/gopherjs/gopherjs/issues/1085")
}
