//go:build js

package big

import "testing"

func TestLinkerGC(t *testing.T) {
	t.Skip("The test is specific to GC's linker.")
}
