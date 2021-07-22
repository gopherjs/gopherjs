//go:build js
// +build js

package js_test

import (
	"testing"

	"github.com/gopherjs/gopherjs/js"
)

func TestInternalizeCircularReference(t *testing.T) {
	// See https://github.com/gopherjs/gopherjs/issues/968.
	js.Global.Call("eval", `
	var issue968a = {};
	var issue968b = {'a': issue968a};
	issue968a.b = issue968b;`)
	_ = js.Global.Get("issue968a").Interface()
}
