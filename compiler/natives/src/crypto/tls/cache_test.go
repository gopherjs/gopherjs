//go:build js

package tls

import "testing"

func TestCertCache(t *testing.T) {
	t.Skip("GC based Cache is not supported by GopherJS")
}

func BenchmarkCertCache(b *testing.B) {
	b.Skip("GC based Cache is not supported by GopherJS")
}
