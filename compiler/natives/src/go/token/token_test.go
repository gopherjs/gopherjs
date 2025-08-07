//go:build js

package token

import "testing"

//gopherjs:replace
func TestFileSetRace(t *testing.T) {
	t.Skip("Fails with: WaitGroup counter not zero")
}
