// +build js

package runtime

import "github.com/gopherjs/gopherjs/js"

const GOOS = theGoos
const GOARCH = "js"
const Compiler = "gopherjs"

// fake for error.go
type eface struct {
	_type *struct {
		_string *string
	}
}

func init() {
	jsPkg := js.Global.Get("$packages").Get("github.com/gopherjs/gopherjs/js")
	js.Global.Set("$jsObjectPtr", jsPkg.Get("Object").Get("ptr"))
	js.Global.Set("$jsErrorPtr", jsPkg.Get("Error").Get("ptr"))
	js.Global.Set("$throwRuntimeError", js.InternalObject(func(msg string) {
		panic(errorString(msg))
	}))
	// avoid dead code elimination
	var e error
	e = &TypeAssertionError{}
	_ = e
}

func GOROOT() string {
	process := js.Global.Get("process")
	if process == js.Undefined {
		return "/"
	}
	goroot := process.Get("env").Get("GOROOT")
	if goroot != js.Undefined {
		return goroot.String()
	}
	return defaultGoroot
}

func Breakpoint() {
	js.Debugger()
}

func Caller(skip int) (pc uintptr, file string, line int, ok bool) {
	info := js.Global.Get("Error").New().Get("stack").Call("split", "\n").Index(skip + 2)
	if info == js.Undefined {
		return 0, "", 0, false
	}
	parts := info.Call("substring", info.Call("indexOf", "(").Int()+1, info.Call("indexOf", ")").Int()).Call("split", ":")
	return 0, parts.Index(0).String(), parts.Index(1).Int(), true
}

func Callers(skip int, pc []uintptr) int {
	return 0
}

func GC() {
}

func Goexit() {
	js.Global.Get("$curGoroutine").Set("exit", true)
	js.Global.Call("$throw", nil)
}

func GOMAXPROCS(n int) int {
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

type MemStats struct {
	// General statistics.
	Alloc      uint64 // bytes allocated and still in use
	TotalAlloc uint64 // bytes allocated (even if freed)
	Sys        uint64 // bytes obtained from system (sum of XxxSys below)
	Lookups    uint64 // number of pointer lookups
	Mallocs    uint64 // number of mallocs
	Frees      uint64 // number of frees

	// Main allocation heap statistics.
	HeapAlloc    uint64 // bytes allocated and still in use
	HeapSys      uint64 // bytes obtained from system
	HeapIdle     uint64 // bytes in idle spans
	HeapInuse    uint64 // bytes in non-idle span
	HeapReleased uint64 // bytes released to the OS
	HeapObjects  uint64 // total number of allocated objects

	// Low-level fixed-size structure allocator statistics.
	//	Inuse is bytes used now.
	//	Sys is bytes obtained from system.
	StackInuse  uint64 // bytes used by stack allocator
	StackSys    uint64
	MSpanInuse  uint64 // mspan structures
	MSpanSys    uint64
	MCacheInuse uint64 // mcache structures
	MCacheSys   uint64
	BuckHashSys uint64 // profiling bucket hash table
	GCSys       uint64 // GC metadata
	OtherSys    uint64 // other system allocations

	// Garbage collector statistics.
	NextGC       uint64 // next collection will happen when HeapAlloc ≥ this amount
	LastGC       uint64 // end time of last collection (nanoseconds since 1970)
	PauseTotalNs uint64
	PauseNs      [256]uint64 // circular buffer of recent GC pause durations, most recent at [(NumGC+255)%256]
	PauseEnd     [256]uint64 // circular buffer of recent GC pause end times
	NumGC        uint32
	EnableGC     bool
	DebugGC      bool

	// Per-size allocation statistics.
	// 61 is NumSizeClasses in the C code.
	BySize [61]struct {
		Size    uint32
		Mallocs uint64
		Frees   uint64
	}
}

func ReadMemStats(m *MemStats) {
}

func SetFinalizer(x, f interface{}) {
}

type Func struct {
	opaque struct{} // unexported field to disallow conversions
}

func (_ *Func) Entry() uintptr                              { return 0 }
func (_ *Func) FileLine(pc uintptr) (file string, line int) { return "", 0 }
func (_ *Func) Name() string                                { return "" }

func FuncForPC(pc uintptr) *Func {
	return nil
}

var MemProfileRate int = 512 * 1024

func SetBlockProfileRate(rate int) {
}

func Stack(buf []byte, all bool) int {
	s := js.Global.Get("Error").New().Get("stack")
	if s == js.Undefined {
		return 0
	}
	return copy(buf, s.Call("substr", s.Call("indexOf", "\n").Int()+1).String())
}

func LockOSThread() {}

func UnlockOSThread() {}

func Version() string {
	return theVersion
}
