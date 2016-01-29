// +build js

package os

import (
	"github.com/gopherjs/gopherjs/js"
)

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

// Override the builtin os.Hostname(). The builtin Hostname() will result in
// a non-working syscall on most (all?) platforms
func Hostname() (string, error) {
	// Browser
	// The browser doesn't know anything about any hostname, so we use the host part of
	// the current url.
	if location := js.Global.Get("location"); location != js.Undefined {
		if browserHostname := location.Get("hostname"); browserHostname != js.Undefined {
			return browserHostname.String(), nil
		}
	}

	// Node.js
	// We call node's require("os").hostname() which should return current hostname
	if require := js.Global.Get("require"); require != js.Undefined {
		// "os" is in nodejs core, it should always exist
		if nodeHostname := require.Invoke("os").Call("hostname"); nodeHostname != js.Undefined {
			return nodeHostname.String(), nil
		}
	}

	return "", ErrNotExist
}
