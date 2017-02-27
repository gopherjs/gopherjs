// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testing

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func runExample(eg InternalExample) (ok bool) {
	if *chatty {
		fmt.Printf("=== RUN   %s\n", eg.Name)
	}

	// Capture stdout.
	stdout := os.Stdout
	w, err := ioutil.TempFile("", "."+eg.Name+".stdout.")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Stdout = w

	start := time.Now()
	ok = true

	// Clean up in a deferred call so we can recover if the example panics.
	defer func() {
		dstr := fmtDuration(time.Now().Sub(start))

		// Close pipe, restore stdout, get output.
		w.Close()
		os.Stdout = stdout
		out, e := ioutil.ReadFile(w.Name())
		if e != nil {
			fmt.Fprintf(os.Stderr, "testing: reading stdout file: %v\n", e)
			os.Exit(1)
		}
		_ = os.Remove(w.Name())

		var fail string
		err := recover()
		got := strings.TrimSpace(string(out))
		want := strings.TrimSpace(eg.Output)
		if eg.Unordered {
			if sortLines(got) != sortLines(want) && err == nil {
				fail = fmt.Sprintf("got:\n%s\nwant (unordered):\n%s\n", out, eg.Output)
			}
		} else {
			if got != want && err == nil {
				fail = fmt.Sprintf("got:\n%s\nwant:\n%s\n", got, want)
			}
		}
		if fail != "" || err != nil {
			fmt.Printf("--- FAIL: %s (%s)\n%s", eg.Name, dstr, fail)
			ok = false
		} else if *chatty {
			fmt.Printf("--- PASS: %s (%s)\n", eg.Name, dstr)
		}
		if err != nil {
			panic(err)
		}
	}()

	// Run example.
	eg.F()
	return
}
