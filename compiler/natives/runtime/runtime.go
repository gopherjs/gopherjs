// +build js

package runtime

import (
	"github.com/gopherjs/gopherjs/js"
)

const theGoarch = "js"

type NotSupportedError struct {
	Feature string
}

func (err *NotSupportedError) Error() string {
	return "not supported by GopherJS: " + err.Feature
}

func init() {
	js.Global.Set("$throwRuntimeError", js.InternalObject(func(msg string) {
		panic(errorString(msg))
	}))
	// avoid dead code elimination
	var e error
	e = &TypeAssertionError{}
	e = &NotSupportedError{}
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

func Breakpoint() {
	js.Debugger()
}

func Caller(skip int) (pc uintptr, file string, line int, ok bool) {
	info := js.Global.Get("Error").New().Get("stack").Call("split", "\n").Index(skip + 2)
	if info.IsUndefined() {
		return 0, "", 0, false
	}
	parts := info.Call("substring", info.Call("indexOf", "(").Int()+1, info.Call("indexOf", ")").Int()).Call("split", ":")
	return 0, parts.Index(0).Str(), parts.Index(1).Int(), true
}

func GC() {
}

func Goexit() {
	js.Global.Get("$curGoroutine").Set("exit", true)
	js.Global.Call("$throw", nil)
}

func GOMAXPROCS(n int) int {
	if n > 1 {
		panic(&NotSupportedError{"GOMAXPROCS > 1"})
	}
	return 1
}

func Gosched() {
	c := make(chan struct{})
	js.Global.Call("setTimeout", func() { close(c) }, 0)
	<-c
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
