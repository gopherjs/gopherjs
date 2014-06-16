// +build js

package os

import (
	"github.com/gopherjs/gopherjs/js"
)

func init() {
	process := js.Global.Get("process")
	if process.IsUndefined() {
		Args = []string{"browser"}
		return
	}
	args := process.Get("argv")
	Args = make([]string, args.Length()-1)
	for i := 0; i < args.Length()-1; i++ {
		Args[i] = args.Index(i + 1).Str()
	}
}
