//go:build js
// +build js

package runtime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gopherjs/gopherjs/js"
)

func Test_parseCallstack(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "Chrome 96.0.4664.110 on Linux",
			input: `at foo (eval at $b (https://gopherjs.github.io/playground/playground.js:102:11836), <anonymous>:25887:60)
	at main (eval at $b (https://gopherjs.github.io/playground/playground.js:102:11836), <anonymous>:25880:8)
	at $init (eval at $b (https://gopherjs.github.io/playground/playground.js:102:11836), <anonymous>:25970:9)
	at $goroutine (eval at $b (https://gopherjs.github.io/playground/playground.js:102:11836), <anonymous>:1602:19)
	at $runScheduled (eval at $b (https://gopherjs.github.io/playground/playground.js:102:11836), <anonymous>:1648:7)
	at $schedule (eval at $b (https://gopherjs.github.io/playground/playground.js:102:11836), <anonymous>:1672:5)
	at $go (eval at $b (https://gopherjs.github.io/playground/playground.js:102:11836), <anonymous>:1634:3)
	at eval (eval at $b (https://gopherjs.github.io/playground/playground.js:102:11836), <anonymous>:25982:1)
	at eval (eval at $b (https://gopherjs.github.io/playground/playground.js:102:11836), <anonymous>:25985:4)
	at eval (<anonymous>)
	at $b (https://gopherjs.github.io/playground/playground.js:102:11836)
	at k.e.$externalizeWrapper.e.$externalizeWrapper [as run] (https://gopherjs.github.io/playground/playground.js:5:30547)
	at https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js:175:190
	at https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js:192:165
	at k.$eval (https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js:112:32)
	at k.$apply (https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js:112:310)
	at HTMLInputElement.<anonymous> (https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js:192:147)
	at https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js:31:225
	at Array.forEach (<anonymous>)
	at q (https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js:7:280)
	at HTMLInputElement.c (https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js:31:207)`,
			want: `foo https://gopherjs.github.io/playground/playground.js 102
main https://gopherjs.github.io/playground/playground.js 102
$init https://gopherjs.github.io/playground/playground.js 102
$goroutine https://gopherjs.github.io/playground/playground.js 102
$runScheduled https://gopherjs.github.io/playground/playground.js 102
$go https://gopherjs.github.io/playground/playground.js 102
eval https://gopherjs.github.io/playground/playground.js 102
eval https://gopherjs.github.io/playground/playground.js 102
eval  0
$b https://gopherjs.github.io/playground/playground.js 102
k.e.$externalizeWrapper.e.$externalizeWrapper [as run] https://gopherjs.github.io/playground/playground.js 5
 at   0
 at   0
k.$eval https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js 112
k.$apply https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js 112
HTMLInputElement.<anonymous> https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js 192
 at   0
Array.forEach  0
q https://ajax.googleapis.com/ajax/libs/angularjs/1.2.18/angular.min.js 7
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
