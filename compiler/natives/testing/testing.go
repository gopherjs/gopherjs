// +build js

package testing

import (
	"flag"
	"fmt"
	"os"
	"runtime"
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

		done := make(chan struct{})
		go func() {
			defer func() {
				err := recover()
				if e, ok := err.(*runtime.NotSupportedError); ok {
					t.log(e.Error())
					t.skip()
					err = nil
				}
				if err != nil {
					t.Fail()
					t.report()
					panic(err)
				}
				close(done)
			}()
			test.F(t) //gopherjs:blocking
		}()
		<-done

		t.common.duration = time.Now().Sub(t.common.start)
		t.report()
		failed = failed || t.common.failed
	}

	if failed {
		os.Exit(1)
	}
	os.Exit(0)
}
