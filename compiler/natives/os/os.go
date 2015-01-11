// +build js

package os

import (
	"github.com/gopherjs/gopherjs/js"
)

func runtime_args() []string { // not called on Windows
	return Args
}

func init() {
	process := js.Global.Get("process")
	if process == js.Undefined {
		Args = []string{"?"}
	}
	argv := process.Get("argv")
	Args = make([]string, argv.Length()-1)
	for i := 0; i < argv.Length()-1; i++ {
		Args[i] = argv.Index(i + 1).Str()
	}
}
