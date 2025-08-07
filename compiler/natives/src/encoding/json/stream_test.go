//go:build js

package json

import "testing"

//gopherjs:replace
func TestHTTPDecoding(t *testing.T) {
	t.Skip("network access is not supported by GopherJS")
}
