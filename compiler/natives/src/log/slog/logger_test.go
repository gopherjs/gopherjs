//go:build js

package slog

import "testing"

//gopherjs:replace
func TestAlloc(t *testing.T) {
	t.Skip("testing.AllocsPerRun not supported in GopherJS")
}
