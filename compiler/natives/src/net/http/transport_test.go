//go:build js

package http_test

import "testing"

//gopherjs:replace
func TestTransportPersistConnLeakNeverIdle(t *testing.T) {
	t.Skip("test relied on runtime.SetFinalizer(), which is not supported by GopherJS.")
}

//gopherjs:replace
func TestTransportPersistConnContextLeakMaxConnsPerHost(t *testing.T) {
	t.Skip("test relied on runtime.SetFinalizer(), which is not supported by GopherJS.")
}
