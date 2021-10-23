//go:build js
// +build js

package http_test

import "testing"

func TestTransportPersistConnLeakNeverIdle(t *testing.T) {
	t.Skip("test relied on runtime.SetFinalizer(), which is not supported by GopherJS.")
}

func TestTransportPersistConnContextLeakMaxConnsPerHost(t *testing.T) {
	t.Skip("test relied on runtime.SetFinalizer(), which is not supported by GopherJS.")
}
