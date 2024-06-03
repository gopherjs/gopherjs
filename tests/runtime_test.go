//go:build js && gopherjs

package tests

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"testing"
	_ "unsafe"

	"github.com/google/go-cmp/cmp"
	"github.com/gopherjs/gopherjs/js"
)

func Test_parseCallFrame(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Chrome 96.0.4664.110 on Linux #1",
			input: "at foo (eval at $b (https://gopherjs.github.io/playground/playground.js:102:11836), <anonymous>:25887:60)",
			want:  "foo https://gopherjs.github.io/playground/playground.js 102 11836",
		},
		{
			name:  "Chrome 96, anonymous eval",
			input: "	at eval (<anonymous>)",
			want:  "eval <anonymous> 0 0",
		},
		{
			name:  "Chrome 96, anonymous Array.forEach",
			input: "	at Array.forEach (<anonymous>)",
			want:  "Array.forEach <anonymous> 0 0",
		},
		{
			name:  "Chrome 96, file location only",
			input: "at https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js:31:225",
			want:  "<none> https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js 31 225",
		},
		{
			name:  "Chrome 96, aliased function",
			input: "at k.e.$externalizeWrapper.e.$externalizeWrapper [as run] (https://gopherjs.github.io/playground/playground.js:5:30547)",
			want:  "run https://gopherjs.github.io/playground/playground.js 5 30547",
		},
		{
			name:  "Node.js v12.22.5",
			input: "    at Script.runInThisContext (vm.js:120:18)",
			want:  "Script.runInThisContext vm.js 120 18",
		},
		{
			name:  "Node.js v12.22.5, aliased function",
			input: "at REPLServer.runBound [as eval] (domain.js:440:12)",
			want:  "eval domain.js 440 12",
		},
		{
			name:  "Firefox 78.15.0esr Linux",
			input: "getEvalResult@resource://devtools/server/actors/webconsole/eval-with-debugger.js:231:24",
			want:  "getEvalResult resource://devtools/server/actors/webconsole/eval-with-debugger.js 231 24",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := js.Global.Get("String").New(tt.input)
			frame := runtime.ParseCallFrame(lines)
			got := fmt.Sprintf("%v %v %v %v", frame.FuncName, frame.File, frame.Line, frame.Col)
			if tt.want != got {
				t.Errorf("Unexpected result: %s", got)
			}
		})
	}
}

func TestBuildPlatform(t *testing.T) {
	if runtime.GOOS != "js" {
		t.Errorf("Got runtime.GOOS=%q. Want: %q.", runtime.GOOS, "js")
	}
	if runtime.GOARCH != "ecmascript" {
		t.Errorf("Got runtime.GOARCH=%q. Want: %q.", runtime.GOARCH, "ecmascript")
	}
}

type funcName string

func masked(_ funcName) funcName { return "<MASKED>" }

type callStack []funcName

func (c *callStack) capture() {
	*c = nil
	pc := [100]uintptr{}
	depth := runtime.Callers(0, pc[:])
	frames := runtime.CallersFrames(pc[:depth])
	for true {
		frame, more := frames.Next()
		*c = append(*c, funcName(frame.Function))
		if !more {
			break
		}
	}
}

func TestCallers(t *testing.T) {
	got := callStack{}

	// Some of the GopherJS function names don't match upstream Go, or even the
	// function names in the Go source when minified.
	// Until https://github.com/gopherjs/gopherjs/issues/1085 is resolved, the
	// mismatch is difficult to avoid, but we can at least use "masked" frames to
	// make sure the number of frames matches expected.
	want := callStack{
		masked("runtime.Callers"),
		masked("github.com/gopherjs/gopherjs/tests.(*callerNames).capture"),
		masked("github.com/gopherjs/gopherjs/tests.TestCallers.func{1,2}"),
		masked("testing.tRunner"),
		"runtime.goexit",
	}

	opts := cmp.Comparer(func(a, b funcName) bool {
		if a == masked("") || b == masked("") {
			return true
		}
		return a == b
	})

	t.Run("Normal", func(t *testing.T) {
		got.capture()
		if diff := cmp.Diff(want, got, opts); diff != "" {
			t.Errorf("runtime.Callers() returned a diff (-want,+got):\n%s", diff)
		}
	})

	t.Run("Deferred", func(t *testing.T) {
		defer func() {
			if diff := cmp.Diff(want, got, opts); diff != "" {
				t.Errorf("runtime.Callers() returned a diff (-want,+got):\n%s", diff)
			}
		}()
		defer got.capture()
	})

	t.Run("Recover", func(t *testing.T) {
		defer func() {
			recover()
			got.capture()

			want := callStack{
				masked("runtime.Callers"),
				masked("github.com/gopherjs/gopherjs/tests.(*callerNames).capture"),
				masked("github.com/gopherjs/gopherjs/tests.TestCallers.func3.1"),
				"runtime.gopanic",
				masked("github.com/gopherjs/gopherjs/tests.TestCallers.func{1,2}"),
				masked("testing.tRunner"),
				"runtime.goexit",
			}
			if diff := cmp.Diff(want, got, opts); diff != "" {
				t.Errorf("runtime.Callers() returned a diff (-want,+got):\n%s", diff)
			}
		}()
		panic("panic")
	})
}

// Need this to tunnel into `internal/godebug` and run a test
// without causing a dependency cycle with the `testing` package.
//
//go:linkname godebug_setUpdate runtime.godebug_setUpdate
func godebug_setUpdate(update func(string, string))

func Test_GoDebugInjection(t *testing.T) {
	buf := []string{}
	update := func(def, env string) {
		if def != `` {
			t.Errorf(`Expected the default value to be empty but got %q`, def)
		}
		buf = append(buf, strconv.Quote(env))
	}
	check := func(want string) {
		if got := strings.Join(buf, `, `); got != want {
			t.Errorf(`Unexpected result: got: %q, want: %q`, got, want)
		}
		buf = buf[:0]
	}

	// Call it multiple times to ensure that the watcher is only injected once.
	// Each one of these calls should emit an update first, then when GODEBUG is set.
	godebug_setUpdate(update)
	godebug_setUpdate(update)
	check(`"", ""`) // two empty strings for initial update calls.

	t.Setenv(`GODEBUG`, `gopherJSTest=ben`)
	check(`"gopherJSTest=ben"`) // must only be once for update for new value.

	godebug_setUpdate(update)
	check(`"gopherJSTest=ben"`) // must only be once for initial update with already set value.

	t.Setenv(`GODEBUG`, `gopherJSTest=tom`)
	t.Setenv(`GODEBUG`, `gopherJSTest=sam`)
	t.Setenv(`NOT_GODEBUG`, `gopherJSTest=bob`)
	check(`"gopherJSTest=tom", "gopherJSTest=sam"`)
}
