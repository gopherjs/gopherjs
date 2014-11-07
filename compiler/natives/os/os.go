// +build js

package os

import (
	"github.com/gopherjs/gopherjs/js"
)

func runtime_args() []string {
	process := js.Global.Get("process")
	if process.IsUndefined() {
		return []string{"browser"}
	}
	argv := process.Get("argv")
	args := make([]string, argv.Length()-1)
	for i := 0; i < argv.Length()-1; i++ {
		args[i] = argv.Index(i + 1).Str()
	}
	return args
}
