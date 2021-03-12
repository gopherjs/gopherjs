// +build js

package os

import (
	"errors"

	"github.com/gopherjs/gopherjs/js"
)

const isBigEndian = false

func runtime_args() []string { // not called on Windows
	return Args
}

func init() {
	if process := js.Global.Get("process"); process != js.Undefined {
		argv := process.Get("argv")
		Args = make([]string, argv.Length()-1)
		for i := 0; i < argv.Length()-1; i++ {
			Args[i] = argv.Index(i + 1).String()
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

func fastrand() uint32 {
	// TODO(nevkontakte): Upstream this function is actually linked to runtime.os_fastrand()
	// via a go:linkname directive, which is currently unsupported by GopherJS.
	// For now we just substitute it with JS's Math.random(), but it is likely slower
	// than the fastrand.
	return uint32(js.Global.Get("Math").Call("random").Float() * (1<<32 - 1))
}
