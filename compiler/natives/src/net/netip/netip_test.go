//go:build js
// +build js

package netip_test

import "testing"

func TestAddrStringAllocs(t *testing.T) {
	t.Skip("testing.AllocsPerRun not supported in GopherJS")
}
