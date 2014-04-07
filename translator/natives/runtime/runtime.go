// +build js

package runtime

import (
	"github.com/gopherjs/gopherjs/js"
)

func getgoroot() string {
	process := js.Global.Get("process")
	if process.IsUndefined() {
		return "/"
	}
	goroot := process.Get("env").Get("GOROOT")
	if goroot.IsUndefined() {
		return ""
	}
	return goroot.Str()
}

func GC() {
}

func GOMAXPROCS(n int) int {
	if n > 1 {
		js.Global.Call("go$notSupported", "GOMAXPROCS > 1")
	}
	return 1
}

func NumCPU() int {
	return 1
}

func ReadMemStats(m *MemStats) {
}

func SetFinalizer(x, f interface{}) {
}
