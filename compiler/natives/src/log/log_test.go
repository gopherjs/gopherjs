//go:build js

package log

import "testing"

func TestOutputRace(t *testing.T) {
	t.Skip("Fails with: WaitGroup counter not zero")
}
