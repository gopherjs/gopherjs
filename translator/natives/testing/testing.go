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
		Main2("", "", nil, nil)
	}
}

func Main2(pkgPath string, dir string, names []string, tests []func(*T)) {
	flag.Parse()
	if len(tests) == 0 {
		fmt.Println("testing: warning: no tests to run")
	}
	d, err := os.Open(dir)
	if err != nil {
		panic(err)
	}
	d.Chdir()
	start := time.Now()
	status := "ok  "
	for i := 0; i < len(tests); i++ {
		t := &T{
			common: common{
				start: time.Now(),
			},
			name: names[i],
		}
		t.self = t
		if *chatty {
			fmt.Printf("=== RUN %s\n", t.name)
		}
		err := runTest(tests[i], t)
		if err != nil {
			switch {
			case !err.Get("go$exit").IsUndefined():
				// test failed or skipped
				err = nil
			case !err.Get("go$notSupported").IsUndefined():
				t.log(err.Get("message").String())
				t.skip()
				err = nil
			default:
				t.Fail()
			}
		}
		t.common.duration = time.Now().Sub(t.common.start)
		t.report()
		if err != nil {
			panic(err)
		}
		if t.common.failed {
			status = "FAIL"
		}
	}
	duration := time.Now().Sub(start)
	fmt.Printf("%s\t%s\t%.3fs\n", status, pkgPath, duration.Seconds())
}

func runTest(func(*T), *T) js.Object
