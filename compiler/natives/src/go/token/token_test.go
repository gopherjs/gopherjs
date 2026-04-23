//go:build js

package token

import "testing"

//gopherjs:replace
func TestFileSetRace(t *testing.T) {
	t.Skip("Fails with: WaitGroup counter not zero")
	// "go/token" files are updated in build/build.go#augmentOriginalImports to
	// use the `gopherjs/nosync` package instead of `sync`.
	// `nosync`'s WaitGroup Wait method panics if the counter is not zero since it assumes
	// the code will be synchronously executed for specific packages including go/token.
}
