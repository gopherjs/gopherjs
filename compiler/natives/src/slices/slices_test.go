//go:build js

package slices

import "testing"

//gopherjs:replace
func TestGrow(t *testing.T) {
	t.Skip("testing.AllocsPerRun not supported in GopherJS")
}
