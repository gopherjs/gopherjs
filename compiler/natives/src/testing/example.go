//go:build js

package testing

import (
	"fmt"
	"os"
	"time"
)

func runExample(eg InternalExample) (ok bool) {
	if chatty.on {
		fmt.Printf("=== RUN   %s\n", eg.Name)
	}

	// Capture stdout.
	stdout := os.Stdout
	w, err := os.CreateTemp("", "."+eg.Name+".stdout.")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Stdout = w

	finished := false
	start := time.Now()

	// Clean up in a deferred call so we can recover if the example panics.
	defer func() {
		timeSpent := time.Since(start)

		// Close file, restore stdout, get output.
		w.Close()
		os.Stdout = stdout
		out, readFileErr := os.ReadFile(w.Name())
		_ = os.Remove(w.Name())
		if readFileErr != nil {
			fmt.Fprintf(os.Stderr, "testing: reading stdout file: %v\n", readFileErr)
			os.Exit(1)
		}

		err := recover()
		ok = eg.processRunResult(string(out), timeSpent, finished, err)
	}()

	// Run example.
	eg.F()
	finished = true
	return
}
