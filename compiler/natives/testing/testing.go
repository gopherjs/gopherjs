// +build js

package testing

import (
	"flag"
	"fmt"
	"github.com/gopherjs/gopherjs/js"
	"os"
	"time"
)

func init() {
	x := false
	if x { // avoid dead code elimination
		Main(nil, nil, nil, nil)
	}
}

func Main(matchString func(pat, str string) (bool, error), tests []InternalTest, benchmarks []InternalBenchmark, examples []InternalExample) {
	flag.Parse()
	if len(tests) == 0 {
		fmt.Println("testing: warning: no tests to run")
	}

	failed := false
	for _, test := range tests {
		t := &T{
			common: common{
				start: time.Now(),
			},
			name: test.Name,
		}
		t.self = t
		if *chatty {
			fmt.Printf("=== RUN %s\n", t.name)
		}
		err := js.Global.Call("$catch", func() {
			test.F(t)
		})
		js.Global.Set("$jsErr", nil)
		if err != nil {
			switch {
			case !err.Get("$exit").IsUndefined():
				// test failed or skipped
				err = nil
			case !err.Get("$notSupported").IsUndefined():
				t.log(err.Get("message").Str())
				t.skip()
				err = nil
			default:
				t.Fail()
			}
		}
		t.common.duration = time.Now().Sub(t.common.start)
		t.report()
		if err != nil {
			js.Global.Call("$throw", js.InternalObject(err))
		}
		failed = failed || t.common.failed
	}

	if failed {
		os.Exit(1)
	}
	os.Exit(0)
}
