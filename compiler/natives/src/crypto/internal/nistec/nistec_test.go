//go:build js
// +build js

package nistec_test

import "testing"

func TestAllocations(t *testing.T) {
	t.Skip("testing.AllocsPerRun not supported in GopherJS")
}
