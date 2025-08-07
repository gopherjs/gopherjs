//go:build js

package netip_test

import "testing"

//gopherjs:replace
func TestAddrStringAllocs(t *testing.T) {
	t.Skip("testing.AllocsPerRun not supported in GopherJS")
}
