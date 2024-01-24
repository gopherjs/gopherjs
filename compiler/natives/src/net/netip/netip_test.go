//go:build js

package netip

import "testing"

func TestAddrStringAllocs(t *testing.T) {
	t.Skip("testing.AllocsPerRun not supported in GopherJS")
}
