//go:build js

package x509

import "testing"

//gopherjs:keep-original
func TestConstraintCases(t *testing.T) {
	if testing.Short() {
		// These tests are slow under GopherJS. Since GopherJS doesn't touch
		// business logic behind them, there's little value in running them all.
		// Instead, in the short mode we just just the first few as a smoke test.
		nameConstraintsTests = nameConstraintsTests[0:5]
	}
	_gopherjs_original_TestConstraintCases(t)
}
