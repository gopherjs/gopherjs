//go:build js
// +build js

package runtime

import (
	"fmt"
	"strings"
	"testing"

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
			want:  "foo https://gopherjs.github.io/playground/playground.js 102",
		},
		{
			name: "Chrome 96, anonymous eval",
			input: "	at eval (<anonymous>)",
			want: "eval <anonymous> 0",
		},
		{
			name: "Chrome 96, anonymous Array.forEach",
			input: "	at Array.forEach (<anonymous>)",
			want: "Array.forEach <anonymous> 0",
		},
		{
			name:  "Chrome 96, file location only",
			input: "at https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js:31:225",
			want:  "<none> https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js 31",
		},
		{
			name:  "Chrome 96, aliased function",
			input: "at k.e.$externalizeWrapper.e.$externalizeWrapper [as run] (https://gopherjs.github.io/playground/playground.js:5:30547)",
			want:  "run https://gopherjs.github.io/playground/playground.js 5",
		},
		{
			name:  "Node.js v12.22.5",
			input: "    at Script.runInThisContext (vm.js:120:18)",
			want:  "Script.runInThisContext vm.js 120",
		},
		{
			name:  "Node.js v12.22.5, aliased function",
			input: "at REPLServer.runBound [as eval] (domain.js:440:12)",
			want:  "eval domain.js 440",
		},
		{
			name:  "Firefox 78.15.0esr Linux",
			input: "getEvalResult@resource://devtools/server/actors/webconsole/eval-with-debugger.js:231:24",
			want:  "getEvalResult resource://devtools/server/actors/webconsole/eval-with-debugger.js 231",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := js.Global.Get("String").New(tt.input)
			frame := parseCallFrame(lines)
			got := fmt.Sprintf("%v %v %v", frame.FuncName, frame.File, frame.Line)
			if tt.want != got {
				t.Errorf("Unexpected result: %s", got)
			}
		})
	}
}

func Test_parseCallstack(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "Chrome 96.0.4664.110 on Linux",
			input: `at foo (eval at $b (https://gopherjs.github.io/playground/playground.js:102:11836), <anonymous>:25887:60)
	at eval (<anonymous>)
	at k.e.$externalizeWrapper.e.$externalizeWrapper [as run] (https://gopherjs.github.io/playground/playground.js:5:30547)
	at HTMLInputElement.<anonymous> (https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js:192:147)
	at HTMLInputElement.c (https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js:31:207)`,
			want: `foo https://gopherjs.github.io/playground/playground.js 102
eval <anonymous> 0
run https://gopherjs.github.io/playground/playground.js 5
HTMLInputElement.<anonymous> https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js 192
HTMLInputElement.c https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js 31`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := js.Global.Get("String").New(tt.input).Call("split", "\n")
			frames := parseCallstack(lines)
			got := make([]string, lines.Length())
			for i, frame := range frames {
				got[i] = fmt.Sprintf("%v %v %v", frame.FuncName, frame.File, frame.Line)
			}
			result := strings.Join(got, "\n")
			if tt.want != result {
				t.Errorf("Unexpected result: %s", result)
			}
		})
	}
}
