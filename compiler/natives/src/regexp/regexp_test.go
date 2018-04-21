// +build js

package regexp

import (
	"testing"
)

//gopherjs:keep_overridden
func TestOnePassCutoff(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Log(r)
			t.Skip("'Maximum call stack size exceeded' may happen on V8, skipping")
		}
	}()

	_gopherjs_overridden_TestOnePassCutoff(t)
}
