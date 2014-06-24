// +build js

package runtime

import (
	"github.com/gopherjs/gopherjs/js"
)

func init() {
	js.Global.Set("$throwRuntimeError", func(msg string) {
		panic(errorString(msg))
	})
	// avoid dead code elimination
	e := TypeAssertionError{}
	_ = e
}

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

func Caller(skip int) (pc uintptr, file string, line int, ok bool) {
	info := js.Global.Call("$getStack").Index(skip + 3)
	if info.IsUndefined() {
		return 0, "", 0, false
	}
	parts := info.Call("substring", info.Call("indexOf", "(").Int()+1, info.Call("indexOf", ")").Int()).Call("split", ":")
	return 0, parts.Index(0).Str(), parts.Index(1).Int(), true
}

func GC() {
}

func Goexit() {
	err := js.Global.Get("Error").New()
	err.Set("$exit", true)
	js.Global.Call("$throw", err)
}

func GOMAXPROCS(n int) int {
	if n > 1 {
		js.Global.Call("$notSupported", "GOMAXPROCS > 1")
	}
	return 1
}

func NumCPU() int {
	return 1
}

func NumGoroutine() int {
	return js.Global.Get("$totalGoroutines").Int()
}

func ReadMemStats(m *MemStats) {
}

func SetFinalizer(x, f interface{}) {
}
