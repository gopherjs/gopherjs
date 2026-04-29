//go:build js

package log

import "testing"

//gopherjs:replace
func TestOutputRace(t *testing.T) {
	t.Skip("Fails with: WaitGroup counter not zero")
	// "log" files are updated in build/build.go#augmentOriginalImports to
	// use the `gopherjs/nosync` package instead of `sync`.
	// `nosync`'s WaitGroup Wait method panics if the counter is not zero since it assumes
	// the code will be synchronously executed for specific packages including log.
}
