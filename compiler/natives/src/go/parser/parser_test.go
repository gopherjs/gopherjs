//go:build js

package parser

import "testing"

//gopherjs:replace
func TestParseDepthLimit(t *testing.T) {
	t.Skip("causes call stack exhaustion on js/ecmascript")
}
