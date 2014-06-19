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

	ok := true
	start := time.Now()
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
		err := runTest.Invoke(js.InternalObject(tests[i]), js.InternalObject(t))
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
			throw.Invoke(js.InternalObject(err))
		}
		ok = ok && !t.common.failed
	}
	duration := time.Now().Sub(start)

	status := "ok  "
	exitCode := 0
	if !ok {
		status = "FAIL"
		exitCode = 1
	}
	fmt.Printf("%s\t%s\t%.3fs\n", status, pkgPath, duration.Seconds())
	os.Exit(exitCode)
}

var runTest = js.Global.Call("eval", `(function(f, t) {
	try {
		f(t);
		return null;
	} catch (e) {
		return e;
	}
})`)

var throw = js.Global.Call("eval", `(function(err) {
	throw err;
})`)
