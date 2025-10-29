package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gopherjs/gopherjs/tests/testdata/jsSourceMap/helper"
)

func main() {
	trace := helper.DoGoThing()
	lines := strings.Split(trace, "\n")

	// Print out part of the stack trace that includes Go code, JS prelude, and .inc.js sources.
	printStackFrame(lines[1]) // doJSThing from .inc.js
	printStackFrame(lines[2]) // helper.DoGoThing from helper.go
	printStackFrame(lines[3]) // main.main from main.go (here)
	printStackFrame(lines[5]) // $goroutine from prelude/goroutines.js
}

var frameSplitter = regexp.MustCompile(`^\s+at\s+(.*)\s+\((.*):(\d+):(\d+)\)$`)

func printStackFrame(line string) {
	matches := frameSplitter.FindStringSubmatch(line)
	method := matches[1]
	if index := strings.LastIndex(method, `/`); index >= 0 {
		// Trim off the package path for readability.
		method = method[index+1:]
	}

	// Normalize path separators to '/' for consistency across platforms.
	file := strings.ReplaceAll(matches[2], `\`, `/`)
	if index := strings.LastIndex(file, "gopherjs"); index >= 0 {
		// Remove the repo root so that this test works for forks as well.
		file = file[index:]
	}
	lineNum := matches[3]
	// colNum := matches[4]
	// ignore the column number for now since for some reason it shifts
	// between minimized and non-minimized builds, e.g. helper.inc.js
	// has column 12 in non-minimized (at front of `Error`)
	// but 20 in minimized (at front of `stack`) builds.
	// This seems to be caused by esbuild and not something we control.
	fmt.Printf("%s\t(%s:%s)\n", method, file, lineNum)
}
