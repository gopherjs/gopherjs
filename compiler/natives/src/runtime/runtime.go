//go:build js
// +build js

package runtime

import (
	"runtime/internal/sys"

	"github.com/gopherjs/gopherjs/js"
)

const GOOS = sys.GOOS
const GOARCH = "js"
const Compiler = "gopherjs"

// The Error interface identifies a run time error.
type Error interface {
	error

	// RuntimeError is a no-op function but
	// serves to distinguish types that are run time
	// errors from ordinary errors: a type is a
	// run time error if it has a RuntimeError method.
	RuntimeError()
}

// TODO(nevkontakte): In the upstream, this struct is meant to be compatible
// with reflect.rtype, but here we use a minimal stub that satisfies the API
// TypeAssertionError expects, which we dynamically instantiate in $assertType().
type _type struct{ str string }

func (t *_type) string() string  { return t.str }
func (t *_type) pkgpath() string { return "" }

// A TypeAssertionError explains a failed type assertion.
type TypeAssertionError struct {
	_interface    *_type
	concrete      *_type
	asserted      *_type
	missingMethod string // one method needed by Interface, missing from Concrete
}

func (*TypeAssertionError) RuntimeError() {}

func (e *TypeAssertionError) Error() string {
	inter := "interface"
	if e._interface != nil {
		inter = e._interface.string()
	}
	as := e.asserted.string()
	if e.concrete == nil {
		return "interface conversion: " + inter + " is nil, not " + as
	}
	cs := e.concrete.string()
	if e.missingMethod == "" {
		msg := "interface conversion: " + inter + " is " + cs + ", not " + as
		if cs == as {
			// provide slightly clearer error message
			if e.concrete.pkgpath() != e.asserted.pkgpath() {
				msg += " (types from different packages)"
			} else {
				msg += " (types from different scopes)"
			}
		}
		return msg
	}
	return "interface conversion: " + cs + " is not " + as +
		": missing method " + e.missingMethod
}

func init() {
	jsPkg := js.Global.Get("$packages").Get("github.com/gopherjs/gopherjs/js")
	js.Global.Set("$jsObjectPtr", jsPkg.Get("Object").Get("ptr"))
	js.Global.Set("$jsErrorPtr", jsPkg.Get("Error").Get("ptr"))
	js.Global.Set("$throwRuntimeError", js.InternalObject(throw))
	buildVersion = js.Global.Get("$goVersion").String()
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
	if v := process.Get("env").Get("GOPHERJS_GOROOT"); v != js.Undefined && v.String() != "" {
		// GopherJS-specific GOROOT value takes precedence.
		return v.String()
	} else if v := process.Get("env").Get("GOROOT"); v != js.Undefined && v.String() != "" {
		return v.String()
	}
	// sys.DefaultGoroot is now gone, can't use it as fallback anymore.
	// TODO: See if a better solution is needed.
	return "/usr/local/go"
}

func Breakpoint() { js.Debugger() }

var (
	// JavaScript runtime doesn't provide access to low-level execution position
	// counters, so we emulate them by recording positions we've encountered in
	// Caller() and Callers() functions and assigning them arbitrary integer values.
	//
	// We use the map and the slice below to convert a "file:line" position
	// into an integer position counter and then to a Func instance.
	knownPositions   = map[string]uintptr{}
	positionCounters = []*Func{}
)

func registerPosition(funcName string, file string, line int) uintptr {
	key := file + ":" + itoa(line)
	if pc, found := knownPositions[key]; found {
		return pc
	}
	f := &Func{
		name: funcName,
		file: file,
		line: line,
	}
	pc := uintptr(len(positionCounters))
	positionCounters = append(positionCounters, f)
	knownPositions[key] = pc
	return pc
}

// itoa converts an integer to a string.
//
// Can't use strconv.Itoa() in the `runtime` package due to a cyclic dependency.
func itoa(i int) string {
	return js.Global.Get("String").New(i).String()
}

// basicFrame contains stack trace information extracted from JS stack trace.
type basicFrame struct {
	FuncName string
	File     string
	Line     int
}

func callstack(skip, limit int) []basicFrame {
	skip = skip + 1 /*skip error message*/ + 1 /*skip callstack's own frame*/
	lines := js.Global.Get("Error").New().Get("stack").Call("split", "\n").Call("slice", skip)
	frames := []basicFrame{}
	l := lines.Get("length").Int()
	for i := 0; i < l && i < limit; i++ {
		info := lines.Index(i)
		pos := info.Call("substring", info.Call("indexOf", "(").Int()+1, info.Call("indexOf", ")").Int())
		parts := pos.Call("split", ":")

		frames = append(frames, basicFrame{
			File:     parts.Index(0).String(),
			Line:     parts.Index(1).Int(),
			FuncName: info.Call("substring", info.Call("indexOf", "at ").Int()+3, info.Call("indexOf", " (").Int()).String(),
		})
	}
	return frames
}

func Caller(skip int) (pc uintptr, file string, line int, ok bool) {
	skip = skip + 1 /*skip Caller's own frame*/
	frames := callstack(skip, 1)
	if len(frames) != 1 {
		return 0, "", 0, false
	}
	pc = registerPosition(frames[0].FuncName, frames[0].File, frames[0].Line)
	return pc, frames[0].File, frames[0].Line, true
}

func Callers(skip int, pc []uintptr) int {
	frames := callstack(skip, len(pc))
	for i, frame := range frames {
		pc[i] = registerPosition(frame.FuncName, frame.File, frame.Line)
	}
	return len(frames)
}

func CallersFrames(callers []uintptr) *Frames {
	result := Frames{}
	for _, pc := range callers {
		fun := FuncForPC(pc)
		result.frames = append(result.frames, Frame{
			PC:       pc,
			Func:     fun,
			Function: fun.name,
			File:     fun.file,
			Line:     fun.line,
			Entry:    fun.Entry(),
		})
	}
	return &result
}

type Frames struct {
	frames  []Frame
	current int
}

func (ci *Frames) Next() (frame Frame, more bool) {
	if ci.current >= len(ci.frames) {
		return Frame{}, false
	}
	f := ci.frames[ci.current]
	ci.current++
	return f, ci.current < len(ci.frames)
}

type Frame struct {
	PC       uintptr
	Func     *Func
	Function string
	File     string
	Line     int
	Entry    uintptr
}

func GC() {}

func Goexit() {
	js.Global.Get("$curGoroutine").Set("exit", true)
	js.Global.Call("$throw", nil)
}

func GOMAXPROCS(int) int { return 1 }

func Gosched() {
	c := make(chan struct{})
	js.Global.Call("$setTimeout", js.InternalObject(func() { close(c) }), 0)
	<-c
}

func NumCPU() int { return 1 }

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
	NextGC        uint64 // next collection will happen when HeapAlloc ≥ this amount
	LastGC        uint64 // end time of last collection (nanoseconds since 1970)
	PauseTotalNs  uint64
	PauseNs       [256]uint64 // circular buffer of recent GC pause durations, most recent at [(NumGC+255)%256]
	PauseEnd      [256]uint64 // circular buffer of recent GC pause end times
	NumGC         uint32
	GCCPUFraction float64 // fraction of CPU time used by GC
	EnableGC      bool
	DebugGC       bool

	// Per-size allocation statistics.
	// 61 is NumSizeClasses in the C code.
	BySize [61]struct {
		Size    uint32
		Mallocs uint64
		Frees   uint64
	}
}

func ReadMemStats(m *MemStats) {
	// TODO(nevkontakte): This function is effectively unimplemented and may
	// lead to silent unexpected behaviors. Consider panicing explicitly.
}

func SetFinalizer(x, f interface{}) {
	// TODO(nevkontakte): This function is effectively unimplemented and may
	// lead to silent unexpected behaviors. Consider panicing explicitly.
}

type Func struct {
	name string
	file string
	line int

	opaque struct{} // unexported field to disallow conversions
}

func (_ *Func) Entry() uintptr { return 0 }

func (f *Func) FileLine(pc uintptr) (file string, line int) {
	if f == nil {
		return "", 0
	}
	return f.file, f.line
}
func (f *Func) Name() string {
	if f == nil || f.name == "" {
		return "<unknown>"
	}
	return f.name
}

func FuncForPC(pc uintptr) *Func {
	ipc := int(pc)
	if ipc >= len(positionCounters) {
		// Since we are faking position counters, the only valid way to obtain one
		// is through a Caller() or Callers() function. If pc is out of positionCounters
		// bounds it must have been obtained in some other way, which is unexpected.
		// If a panic proves problematic, we can return a nil *Func, which will
		// present itself as a generic "unknown" function.
		panic("GopherJS: pc=" + itoa(ipc) + " is out of range of known position counters")
	}
	return positionCounters[ipc]
}

var MemProfileRate int = 512 * 1024

func SetBlockProfileRate(rate int) {
}

func SetMutexProfileFraction(rate int) int {
	// TODO: Investigate this. If it's possible to implement, consider doing so, otherwise remove this comment.
	return 0
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

var buildVersion string // Set by init()

func Version() string {
	return buildVersion
}

func StartTrace() error { return nil }
func StopTrace()        {}
func ReadTrace() []byte

// We fake a cgo environment to catch errors. Therefor we have to implement this and always return 0
func NumCgoCall() int64 {
	return 0
}

func KeepAlive(interface{}) {}

// An errorString represents a runtime error described by a single string.
type errorString string

func (e errorString) RuntimeError() {}

func (e errorString) Error() string {
	return "runtime error: " + string(e)
}

func throw(s string) {
	panic(errorString(s))
}

func nanotime() int64 {
	return js.Global.Get("Date").New().Call("getTime").Int64() * int64(1000_000)
}
