// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testing

import (
	"bytes"
	"fmt"
	"io"
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
	r, w, err := os.Pipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Stdout = w
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		r.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "testing: copying pipe: %v\n", err)
			os.Exit(1)
		}
		outC <- buf.String()
	}()

	start := time.Now()
	ok = true

	// Clean up in a deferred call so we can recover if the example panics.
	defer func() {
		dstr := fmtDuration(time.Now().Sub(start))

		// Close pipe, restore stdout, get output.
		w.Close()
		os.Stdout = stdout
		out := <-outC

		var fail string
		err := recover()
		got := strings.TrimSpace(out)
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
