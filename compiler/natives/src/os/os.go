//go:build js
// +build js

package os

import (
	"errors"
	_ "unsafe" // for go:linkname

	"github.com/gopherjs/gopherjs/js"
)

const isBigEndian = false

func runtime_args() []string { // not called on Windows
	return Args
}

func init() {
	if process := js.Global.Get("process"); process != js.Undefined {
		if argv := process.Get("argv"); argv != js.Undefined {
			Args = make([]string, argv.Length()-1)
			for i := 0; i < argv.Length()-1; i++ {
				Args[i] = argv.Index(i + 1).String()
			}
		}
	}
	if len(Args) == 0 {
		Args = []string{"?"}
	}
}

func runtime_beforeExit() {}

func executable() (string, error) {
	return "", errors.New("Executable not implemented for GOARCH=js")
}

//go:linkname fastrand runtime.fastrand
func fastrand() uint32
