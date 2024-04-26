//go:build ignore
// +build ignore

// skip

// Copyright 2012 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Run runs tests in the test directory.
//
// To run manually with summary, verbose output, and full stack traces of of known failures:
//
//	go run run.go -summary -v -show_known_fails
//
// TODO(bradfitz): docs of some sort, once we figure out how we're changing
// headers of files
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/build/constraint"
	"hash/fnv"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	gbuild "github.com/gopherjs/gopherjs/build"
)

// -----------------------------------------------------------------------------
// GOPHERJS: Known test fails for GopherJS compiler.
//
// TODO: Reduce these to zero or as close as possible.
var knownFails = map[string]failReason{
	"fixedbugs/bug114.go":     {desc: "fixedbugs/bug114.go:15:27: B32 (untyped int constant 4294967295) overflows int"},
	"fixedbugs/bug242.go":     {desc: "bad map check 13 false false Error: fail"},
	"fixedbugs/bug260.go":     {desc: "maybe unsupportedFeature, pointer arithm"},
	"fixedbugs/bug262.go":     {desc: "Error: fail"},
	"fixedbugs/bug273.go":     {desc: "BUG: didn't crash:  badcap1"},
	"fixedbugs/bug328.go":     {desc: "incorrect output"},
	"fixedbugs/bug347.go":     {desc: "BUG: bug347: cannot find caller"},
	"fixedbugs/bug348.go":     {desc: "BUG: bug348: cannot find caller"},
	"fixedbugs/bug352.go":     {desc: "BUG: bug352 struct{}"},
	"fixedbugs/bug409.go":     {desc: "1 2 3 4"},
	"fixedbugs/bug433.go":     {desc: "Error: [object Object]"},
	"fixedbugs/issue11656.go": {desc: "Error: Native function not implemented: runtime/debug.setPanicOnFault"},
	"fixedbugs/issue4085b.go": {desc: "Error: got panic JavaScript error: Invalid typed array length, want len out of range"},
	"fixedbugs/issue4316.go":  {desc: "Error: runtime error: invalid memory address or nil pointer dereference"},
	"fixedbugs/issue4562.go":  {desc: "Error: cannot find issue4562.go on stack"},
	"fixedbugs/issue4620.go":  {desc: "map[0:1 1:2], Error: m[i] != 2"},
	"fixedbugs/issue5856.go":  {category: requiresSourceMapSupport},
	"fixedbugs/issue6899.go":  {desc: "incorrect output -0"},
	"fixedbugs/issue7550.go":  {category: neverTerminates, desc: "FATAL ERROR: invalid table size Allocation failed - process out of memory"},
	"fixedbugs/issue7690.go":  {desc: "Error: runtime error: slice bounds out of range"},
	"fixedbugs/issue8047b.go": {desc: "Error: [object Object]"},

	// These are new tests in Go 1.7.
	"fixedbugs/issue14646.go": {category: unsureIfGopherJSSupportsThisFeature, desc: "tests runtime.Caller behavior in a deferred func in SSA backend... does GopherJS even support runtime.Caller?"},
	"fixedbugs/issue15039.go": {desc: "valid bug but deal with after Go 1.7 support is out? it's likely not a regression"},
	"fixedbugs/issue15281.go": {desc: "also looks valid but deal with after Go 1.7 support is out? it's likely not a regression"},

	// These are new tests in Go 1.8.
	"fixedbugs/issue17381.go": {category: unsureIfGopherJSSupportsThisFeature, desc: "tests runtime.{Callers,FuncForPC} behavior in a deferred func with garbage on stack... does GopherJS even support runtime.{Callers,FuncForPC}?"},
	"fixedbugs/issue18149.go": {desc: "//line directives with filenames are not correctly parsed, see https://github.com/gopherjs/gopherjs/issues/553."},

	// These are new tests in Go 1.9.
	"fixedbugs/issue19182.go": {category: neverTerminates, desc: "needs GOMAXPROCS=2"},
	"fixedbugs/issue19040.go": {desc: `panicwrap error text:
			"runtime error: invalid memory address or nil pointer dereference"
		want:
			"value method main.T.F called using nil *T pointer"`},
	"fixedbugs/issue19246.go": {desc: "expected nil pointer dereference panic"}, // Issue https://golang.org/issues/19246: Failed to evaluate some zero-sized values when converting them to interfaces.

	// These are new tests in Go 1.10.
	"fixedbugs/issue21879.go": {desc: "incorrect output related to runtime.Callers, runtime.CallersFrames, etc."},
	"fixedbugs/issue21887.go": {desc: "incorrect output (although within spec, not worth fixing) for println(^uint64(0)). got: { '$high': 4294967295, '$low': 4294967295, '$val': [Circular] } want: 18446744073709551615"},
	"fixedbugs/issue23305.go": {desc: "GopherJS fails to compile println(0xffffffff), maybe because 32-bit arch"},

	// These are new tests in Go 1.11.
	"fixedbugs/issue21221.go": {category: usesUnsupportedPackage, desc: "uses unsafe package and compares nil pointers"},
	"fixedbugs/issue22662.go": {desc: "line directives not fully working. Error: got /private/var/folders/b8/66r1c5856mqds1mrf2tjtq8w0000gn/T:1; want ??:1"},
	"fixedbugs/issue23188.go": {desc: "incorrect order of evaluation of index operations"},
	"fixedbugs/issue24547.go": {desc: "incorrect computing method sets with shadowed methods"},

	// These are new tests in Go 1.12.
	"fixedbugs/issue23837.go":  {desc: "missing panic on nil pointer-to-empty-struct dereference"},
	"fixedbugs/issue27201.go":  {desc: "incorrect stack trace for nil dereference in inlined function"},
	"fixedbugs/issue27518b.go": {desc: "sigpanic can make dead pointer live again"},
	"fixedbugs/issue29190.go":  {desc: "append does not fail when length overflows", category: neverTerminates},

	// These are new tests in Go 1.12.9.
	"fixedbugs/issue30977.go": {category: neverTerminates, desc: "does for { runtime.GC() }"},
	"fixedbugs/issue32477.go": {category: notApplicable, desc: "uses runtime.SetFinalizer and runtime.GC"},

	// These are new tests in Go 1.13-1.16.
	"fixedbugs/issue19113.go":  {category: lowLevelRuntimeDifference, desc: "JavaScript bit shifts by negative amount don't cause an exception"},
	"fixedbugs/issue24491a.go": {category: notApplicable, desc: "tests interaction between unsafe and GC; uses runtime.SetFinalizer()"},
	"fixedbugs/issue24491b.go": {category: notApplicable, desc: "tests interaction between unsafe and GC; uses runtime.SetFinalizer()"},
	"fixedbugs/issue29504.go":  {category: notApplicable, desc: "requires source map support beyond what GopherJS currently provides"},
	// This test incorrectly passes because main function's name is returned as "main" and not "main.main". Even number of bugs cancel each other out ¯\_(ツ)_/¯
	// "fixedbugs/issue29735.go":  {category: usesUnsupportedPackage, desc: "GopherJS only supports runtime.FuncForPC() with position counters previously returned by runtime.Callers() or runtime.Caller()"},
	"fixedbugs/issue30116.go":  {desc: "GopherJS doesn't specify the array/slice index selector in the out-of-bounds message"},
	"fixedbugs/issue30116u.go": {desc: "GopherJS doesn't specify the array/slice index selector in the out-of-bounds message"},
	"fixedbugs/issue34395.go":  {category: neverTerminates, desc: "https://github.com/gopherjs/gopherjs/issues/1007"},
	"fixedbugs/issue35027.go":  {category: usesUnsupportedPackage, desc: "uses unsupported conversion to reflect.SliceHeader and -gcflags=-d=checkptr"},
	"fixedbugs/issue35576.go":  {category: lowLevelRuntimeDifference, desc: "GopherJS print/println format for floats differs from Go's"},
	"fixedbugs/issue40917.go":  {category: notApplicable, desc: "uses pointer arithmetic and unsupported flag -gcflags=-d=checkptr"},

	// These are new tests in Go 1.17
	"fixedbugs/issue45045.go": {category: notApplicable, desc: "GC related, not relevant to GopherJS"},
	"fixedbugs/issue5493.go":  {category: notApplicable, desc: "GC related, not relevant to GopherJS"},
	"fixedbugs/issue46725.go": {category: notApplicable, desc: "GC related, not relevant to GopherJS"},
	"fixedbugs/issue43444.go": {category: lowLevelRuntimeDifference, desc: "GopherJS println format is different from Go's"},
	"fixedbugs/issue23017.go": {desc: "https://github.com/gopherjs/gopherjs/issues/1063"},

	// These are new tests in Go 1.17.8
	"fixedbugs/issue50854.go": {category: lowLevelRuntimeDifference, desc: "negative int32 overflow behaves differently in JS"},

	// These are new tests in Go 1.18
	"fixedbugs/issue47928.go":  {category: notApplicable, desc: "//go:nointerface is a part of GOEXPERIMENT=fieldtrack and is not supported by GopherJS"},
	"fixedbugs/issue48536.go":  {category: usesUnsupportedPackage, desc: "https://github.com/gopherjs/gopherjs/issues/1130"},
	"fixedbugs/issue48898.go":  {category: other, desc: "https://github.com/gopherjs/gopherjs/issues/1128"},
	"fixedbugs/issue53600.go":  {category: lowLevelRuntimeDifference, desc: "GopherJS println format is different from Go's"},
	"typeparam/chans.go":       {category: neverTerminates, desc: "uses runtime.SetFinalizer() and runtime.GC()."},
	"typeparam/typeswitch5.go": {category: lowLevelRuntimeDifference, desc: "GopherJS println format is different from Go's"},

	// Failures related to the lack of generics support. Ideally, this section
	// should be emptied once https://github.com/gopherjs/gopherjs/issues/1013 is
	// fixed.
	"typeparam/nested.go": {category: usesUnsupportedGenerics, desc: "incomplete support for generic types inside generic functions"},

	// These are new tests in Go 1.19
	"typeparam/issue51521.go": {category: lowLevelRuntimeDifference, desc: "different panic message when calling a method on nil interface"},
	"fixedbugs/issue50672.go": {category: other, desc: "https://github.com/gopherjs/gopherjs/issues/1271"},
	"fixedbugs/issue53653.go": {category: lowLevelRuntimeDifference, desc: "GopherJS println format of int64 is different from Go's"},

	// These are new tests in Go 1.20
	"fixedbugs/issue25897a.go": {category: neverTerminates, desc: "does for { runtime.GC() }"},
	"fixedbugs/issue54343.go":  {category: notApplicable, desc: "uses runtime.SetFinalizer() and runtime.GC()."},
	"fixedbugs/issue57823.go":  {category: notApplicable, desc: "uses runtime.SetFinalizer() and runtime.GC()."},
	"fixedbugs/issue59293.go":  {category: usesUnsupportedPackage, desc: "uses unsafe.SliceData() and unsafe.StringData()."},
	"fixedbugs/issue43942.go":  {category: other, desc: "https://github.com/gopherjs/gopherjs/issues/1126"},
}

type failCategory uint8

const (
	other                    failCategory = iota
	neverTerminates                       // Test never terminates (so avoid starting it).
	usesUnsupportedPackage                // Test fails because it imports an unsupported package, e.g., "unsafe".
	requiresSourceMapSupport              // Test fails without source map support (as configured in CI), because it tries to check filename/line number via runtime.Caller.
	usesUnsupportedGenerics               // Test uses generics (type parameters) that are not currently supported.
	compilerPanic
	unsureIfGopherJSSupportsThisFeature
	lowLevelRuntimeDifference // JavaScript runtime behaves differently from Go in ways that are difficult to work around.
	notApplicable             // Test that doesn't need to run under GopherJS; it doesn't apply to the Go language in a general way.
)

type failReason struct {
	category failCategory
	desc     string
}

// -----------------------------------------------------------------------------

var (
	verbose        = flag.Bool("v", false, "verbose. if set, parallelism is set to 1.")
	numParallel    = flag.Int("n", runtime.NumCPU(), "number of parallel tests to run")
	summary        = flag.Bool("summary", false, "show summary of results")
	showSkips      = flag.Bool("show_skips", false, "show skipped tests")
	showKnownFails = flag.Bool("show_known_fails", false, "show full error output of known fails")
	updateErrors   = flag.Bool("update_errors", false, "update error messages in test file based on compiler output")
	runoutputLimit = flag.Int("l", defaultRunOutputLimit(), "number of parallel runoutput tests to run")

	shard  = flag.Int("shard", 0, "shard index to run. Only applicable if -shards is non-zero.")
	shards = flag.Int("shards", 0, "number of shards. If 0, all tests are run. This is used by the continuous build.")
)

var (
	goos, goarch string

	// dirs are the directories to look for *.go files in.
	// TODO(bradfitz): just use all directories?
	dirs = []string{".", "ken", "chan", "interface", "syntax", "dwarf", "fixedbugs", "typeparam"}

	// ratec controls the max number of tests running at a time.
	ratec chan bool

	// toRun is the channel of tests to run.
	// It is nil until the first test is started.
	toRun chan *test

	// rungatec controls the max number of runoutput tests
	// executed in parallel as they can each consume a lot of memory.
	rungatec chan bool
)

// maxTests is an upper bound on the total number of tests.
// It is used as a channel buffer size to make sure sends don't block.
const maxTests = 5000

func main() {
	flag.Parse()

	// GOPHERJS.
	err := os.Chdir(filepath.Join(gbuild.DefaultGOROOT, "test"))
	if err != nil {
		log.Fatalln(err)
	}

	// GOPHERJS: We're running this script natively, but the tests are executed with js architecture.
	goos = getenv("GOOS", "js")
	goarch = getenv("GOARCH", "ecmascript")

	findExecCmd()

	// Disable parallelism if using a simulator.
	// Do not disable parallelism in verbose mode, since Go's file IO had internal
	// r/w locking, which should make significant output garbling very unlikely.
	// GopherJS CI setup runs these tests in verbose mode, but it can benefit from
	// parallelism a lot.
	if len(findExecCmd()) > 0 {
		*numParallel = 1
	}

	if *verbose {
		fmt.Printf("goos: %q, goarch: %q\n", goos, goarch)
		fmt.Printf("parallel: %d\n", *numParallel)
	}

	ratec = make(chan bool, *numParallel)
	rungatec = make(chan bool, *runoutputLimit)

	var tests []*test
	if flag.NArg() > 0 {
		for _, arg := range flag.Args() {
			if arg == "-" || arg == "--" {
				// Permit running:
				// $ go run run.go - env.go
				// $ go run run.go -- env.go
				// $ go run run.go - ./fixedbugs
				// $ go run run.go -- ./fixedbugs
				continue
			}
			if fi, err := os.Stat(arg); err == nil && fi.IsDir() {
				for _, baseGoFile := range goFiles(arg) {
					tests = append(tests, startTest(arg, baseGoFile))
				}
			} else if strings.HasSuffix(arg, ".go") {
				dir, file := filepath.Split(arg)
				tests = append(tests, startTest(dir, file))
			} else {
				log.Fatalf("can't yet deal with non-directory and non-go file %q", arg)
			}
		}
	} else {
		for _, dir := range dirs {
			for _, baseGoFile := range goFiles(dir) {
				tests = append(tests, startTest(dir, baseGoFile))
			}
		}
	}

	failed := false
	resCount := map[string]int{}
	for _, test := range tests {
		<-test.donec
		// GOPHERJS.
		if test.action == "skip" && !*showSkips {
			continue
		}
		status := "ok  "
		errStr := ""
		// GOPHERJS.
		if _, ok := knownFails[filepath.ToSlash(test.goFileName())]; ok && test.err != nil {
			errStr = test.err.Error()
			test.err = nil
			status = "knfl" // knfl means known failure. Expect test to fail.
		} else if ok && test.err == nil {
			// unok means unexpected okay. Test was expected to fail, but it unexpectedly succeeded.
			// If this is not an accident, it should be removed from knownFails map.
			status = "unok"
		}
		if _, isSkip := test.err.(skipError); isSkip {
			test.err = nil
			errStr = "unexpected skip for " + path.Join(test.dir, test.gofile) + ": " + errStr
			status = "FAIL"
		}
		if test.err != nil {
			status = "FAIL"
			errStr = test.err.Error()
		}
		if status == "FAIL" {
			failed = true
		}
		// GOPHERJS.
		if status == "unok" {
			failed = true
		}
		resCount[status]++
		if status == "skip" && !*verbose && !*showSkips {
			continue
		}
		dt := fmt.Sprintf("%.3fs", test.dt.Seconds())
		if status == "FAIL" {
			fmt.Printf("# go run run.go -- %s\n%s\nFAIL\t%s\t%s\n",
				path.Join(test.dir, test.gofile),
				errStr, test.goFileName(), dt)
			continue
		}
		// GOPHERJS.
		if status == "knfl" && *showKnownFails {
			fmt.Printf("# go run run.go -show_known_fails -- %s\n%s\nknfl\t%s\t%s\n",
				path.Join(test.dir, test.gofile),
				errStr, test.goFileName(), dt)
			continue
		}
		if !*verbose && status != "unok" {
			continue
		}
		fmt.Printf("%s\t%s\t%s\n", status, test.goFileName(), dt)
	}

	if *summary {
		for k, v := range resCount {
			fmt.Printf("%5d %s\n", v, k)
		}
	}

	if failed {
		os.Exit(1)
	}
}

func shardMatch(name string) bool {
	if *shards == 0 {
		return true
	}
	h := fnv.New32()
	io.WriteString(h, name)
	return int(h.Sum32()%uint32(*shards)) == *shard
}

func goFiles(dir string) []string {
	f, err := os.Open(dir)
	check(err)
	dirnames, err := f.Readdirnames(-1)
	f.Close()
	check(err)
	names := []string{}
	for _, name := range dirnames {
		if !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go") && shardMatch(name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

type runCmd func(...string) ([]byte, error)

func compileFile(runcmd runCmd, longname string) (out []byte, err error) {
	return runcmd("go", "tool", "compile", "-e", longname)
}

func compileInDir(runcmd runCmd, dir string, names ...string) (out []byte, err error) {
	cmd := []string{"go", "tool", "compile", "-e", "-D", ".", "-I", "."}
	for _, name := range names {
		cmd = append(cmd, filepath.Join(dir, name))
	}
	return runcmd(cmd...)
}

func linkFile(runcmd runCmd, goname string) (err error) {
	pfile := strings.Replace(goname, ".go", ".o", -1)
	_, err = runcmd("go", "tool", "link", "-w", "-o", "a.exe", "-L", ".", pfile)
	return
}

// skipError describes why a test was skipped.
type skipError string

func (s skipError) Error() string { return string(s) }

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// test holds the state of a test.
type test struct {
	dir, gofile string
	donec       chan bool // closed when done
	dt          time.Duration

	src    string
	action string // "compile", "build", etc.

	tempDir string
	err     error
}

// startTest
func startTest(dir, gofile string) *test {
	t := &test{
		dir:    dir,
		gofile: gofile,
		donec:  make(chan bool, 1),
	}
	if toRun == nil {
		toRun = make(chan *test, maxTests)
		go runTests()
	}
	select {
	case toRun <- t:
	default:
		panic("toRun buffer size (maxTests) is too small")
	}
	return t
}

// runTests runs tests in parallel, but respecting the order they
// were enqueued on the toRun channel.
func runTests() {
	for {
		ratec <- true
		t := <-toRun
		go func() {
			t.run()
			<-ratec
		}()
	}
}

var cwd, _ = os.Getwd()

func (t *test) goFileName() string {
	return filepath.Join(t.dir, t.gofile)
}

func (t *test) goDirName() string {
	return filepath.Join(t.dir, strings.Replace(t.gofile, ".go", ".dir", -1))
}

func goDirFiles(longdir string) (filter []os.DirEntry, err error) {
	files, dirErr := os.ReadDir(longdir)
	if dirErr != nil {
		return nil, dirErr
	}
	for _, gofile := range files {
		if filepath.Ext(gofile.Name()) == ".go" {
			filter = append(filter, gofile)
		}
	}
	return
}

var packageRE = regexp.MustCompile(`(?m)^package (\w+)`)

func goDirPackages(longdir string) ([][]string, error) {
	files, err := goDirFiles(longdir)
	if err != nil {
		return nil, err
	}
	var pkgs [][]string
	m := make(map[string]int)
	for _, file := range files {
		name := file.Name()
		data, err := os.ReadFile(filepath.Join(longdir, name))
		if err != nil {
			return nil, err
		}
		pkgname := packageRE.FindStringSubmatch(string(data))
		if pkgname == nil {
			return nil, fmt.Errorf("cannot find package name in %s", name)
		}
		i, ok := m[pkgname[1]]
		if !ok {
			i = len(pkgs)
			pkgs = append(pkgs, nil)
			m[pkgname[1]] = i
		}
		pkgs[i] = append(pkgs[i], name)
	}
	return pkgs, nil
}

type context struct {
	GOOS   string
	GOARCH string
}

// shouldTest looks for build tags in a source file and returns
// whether the file should be used according to the tags.
func shouldTest(src string, goos, goarch string) (ok bool, whyNot string) {
	// Custom rule, treat js as equivalent to nacl.
	if goarch == "js" {
		goarch = "nacl"
	}

	for _, line := range strings.Split(src, "\n") {
		if strings.HasPrefix(line, "package ") {
			break
		}
		if expr, err := constraint.Parse(line); err == nil {
			ctxt := &context{
				GOOS:   goos,
				GOARCH: goarch,
			}
			if !expr.Eval(ctxt.match) {
				return false, line
			}
		}
	}
	return true, ""
}

func (ctxt *context) match(name string) bool {
	if name == "" {
		return false
	}

	// Tags must be letters, digits, underscores or dots.
	// Unlike in Go identifiers, all digits are fine (e.g., "386").
	for _, c := range name {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' && c != '.' {
			return false
		}
	}

	// GOPHERJS: Ignore "goexperiment." for now
	// GOPHERJS: Don't match "cgo" since not supported
	// GOPHERJS: Don't match "gc"
	if name == ctxt.GOOS || name == ctxt.GOARCH {
		return true
	}

	// GOPHERJS: Don't match "gcflags_noopt"
	if name == "test_run" {
		return true
	}

	return false
}

func init() { checkShouldTest() }

var errTimeout = errors.New("command exceeded time limit")

// run runs a test.
func (t *test) run() {
	start := time.Now()
	defer func() {
		t.dt = time.Since(start)
		close(t.donec)
	}()

	// GOPHERJS: Some tests may never terminate once started. Avoid starting them.
	if kf, ok := knownFails[filepath.ToSlash(t.goFileName())]; ok && kf.category == neverTerminates {
		t.err = skipError("skipping because it doesn't terminate")
		return
	}

	srcBytes, err := os.ReadFile(t.goFileName())
	if err != nil {
		t.err = err
		return
	}
	t.src = string(srcBytes)
	if t.src[0] == '\n' {
		t.err = skipError("starts with newline")
		return
	}

	// Execution recipe stops at first blank line.
	action, _, ok := strings.Cut(t.src, "\n\n")
	if !ok {
		t.err = fmt.Errorf("double newline ending execution recipe not found in %s", t.goFileName())
		return
	}
	if firstLine, rest, ok := strings.Cut(action, "\n"); ok && strings.Contains(firstLine, "+build") {
		// skip first line
		action = rest
	}
	action = strings.TrimPrefix(action, "//")

	// Check for build constraints only up to the actual code.
	header, _, ok := strings.Cut(t.src, "\npackage")
	if !ok {
		header = action // some files are intentionally malformed
	}
	if ok, why := shouldTest(header, goos, goarch); !ok {
		t.action = "skip"
		if *showSkips {
			fmt.Printf("%-20s %-20s: %s\n", t.action, t.goFileName(), why)
		}
		return
	}

	var args, flags []string
	var tim int
	wantError := false
	f, err := splitQuoted(action)
	if err != nil {
		t.err = fmt.Errorf("invalid test recipe: %v", err)
		return
	}
	if len(f) > 0 {
		action = f[0]
		args = f[1:]
	}

	// GOPHERJS: For now, only run with "run", "cmpout" actions, in "fixedbugs" and "typeparam" dirs. Skip all others.
	switch action {
	case "run", "cmpout":
		if d := filepath.Clean(t.dir); d != "fixedbugs" && d != "typeparam" {
			action = "skip"
		}
	default:
		action = "skip"
	}

	switch action {
	case "rundircmpout":
		action = "rundir"
		t.action = "rundir"
	case "cmpout":
		action = "run" // the run case already looks for <dir>/<test>.out files
		fallthrough
	case "compile", "compiledir", "build", "run", "runoutput", "rundir":
		t.action = action
	case "errorcheck", "errorcheckdir", "errorcheckoutput":
		t.action = action
		wantError = true
	case "skip":
		t.action = "skip"
		return
	default:
		t.err = skipError("skipped; unknown pattern: " + action)
		t.action = "??"
		return
	}

	// collect flags
	for len(args) > 0 && strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "-1":
			wantError = true
		case "-0":
			wantError = false
		case "-s":
			// GOPHERJS: Doesn't use singlefilepkgs in test yet.
		case "-t": // timeout in seconds
			args = args[1:]
			var err error
			tim, err = strconv.Atoi(args[0])
			if err != nil {
				t.err = fmt.Errorf("need number of seconds for -t timeout, got %s instead", args[0])
			}
			if s := os.Getenv("GO_TEST_TIMEOUT_SCALE"); s != "" {
				timeoutScale, err := strconv.Atoi(s)
				if err != nil {
					log.Fatalf("failed to parse $GO_TEST_TIMEOUT_SCALE = %q as integer: %v", s, err)
				}
				tim *= timeoutScale
			}
		case "-goexperiment": // set GOEXPERIMENT environment
			args = args[1:]
			// GOPHERJS: Ignore GOEXPERIMENT for now
		default:
			flags = append(flags, args[0])
		}
		args = args[1:]
	}

	t.makeTempDir()
	defer os.RemoveAll(t.tempDir)

	err = os.WriteFile(filepath.Join(t.tempDir, t.gofile), srcBytes, 0o644)
	check(err)

	// A few tests (of things like the environment) require these to be set.
	if os.Getenv("GOOS") == "" {
		os.Setenv("GOOS", goos)
	}
	if os.Getenv("GOARCH") == "" {
		os.Setenv("GOARCH", goarch)
	}

	{
		// GopherJS: we don't support any of -gcflags, but for the most part they
		// are not too relevant to the outcome of the test.
		supportedArgs := []string{}
		for _, a := range args {
			if strings.HasPrefix(a, "-gcflags") {
				continue
			}
			supportedArgs = append(supportedArgs, a)
		}
		args = supportedArgs
	}

	useTmp := true
	runcmd := func(args ...string) ([]byte, error) {
		cmd := exec.Command(args[0], args[1:]...)
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		if useTmp {
			cmd.Dir = t.tempDir
			cmd.Env = envForDir(cmd.Dir)
		}

		var err error
		if tim != 0 {
			err = cmd.Start()
			// This command-timeout code adapted from cmd/go/test.go
			// Note: the Go command uses a more sophisticated timeout
			// strategy, first sending SIGQUIT (if appropriate for the
			// OS in question) to try to trigger a stack trace, then
			// finally much later SIGKILL. If timeouts prove to be a
			// common problem here, it would be worth porting over
			// that code as well. See https://do.dev/issue/50973
			// for more discussion.
			if err == nil {
				tick := time.NewTimer(time.Duration(tim) * time.Second)
				done := make(chan error)
				go func() {
					done <- cmd.Wait()
				}()
				select {
				case err = <-done:
					// ok
				case <-tick.C:
					cmd.Process.Signal(os.Interrupt)
					time.Sleep(1 * time.Second)
					cmd.Process.Kill()
					<-done
					err = errTimeout
				}
				tick.Stop()
			}
		} else {
			err = cmd.Run()
		}
		if err != nil && err != errTimeout {
			err = fmt.Errorf("%s\n%s", err, buf.Bytes())
		}
		return buf.Bytes(), err
	}

	long := filepath.Join(cwd, t.goFileName())
	switch action {
	default:
		t.err = fmt.Errorf("unimplemented action %q", action)

	case "errorcheck":
		cmdline := []string{"go", "tool", "compile", "-e", "-o", "a.o"}
		cmdline = append(cmdline, flags...)
		cmdline = append(cmdline, long)
		out, err := runcmd(cmdline...)
		if wantError {
			if err == nil {
				t.err = fmt.Errorf("compilation succeeded unexpectedly\n%s", out)
				return
			}
			if err == errTimeout {
				t.err = fmt.Errorf("compilation timed out")
				return
			}
		} else {
			if err != nil {
				t.err = err
				return
			}
		}
		if *updateErrors {
			t.updateErrors(string(out), long)
		}
		t.err = t.errorCheck(string(out), long, t.gofile)
		return

	case "compile":
		_, t.err = compileFile(runcmd, long)

	case "compiledir":
		// Compile all files in the directory in lexicographic order.
		longdir := filepath.Join(cwd, t.goDirName())
		pkgs, err := goDirPackages(longdir)
		if err != nil {
			t.err = err
			return
		}
		for _, gofiles := range pkgs {
			_, t.err = compileInDir(runcmd, longdir, gofiles...)
			if t.err != nil {
				return
			}
		}

	case "errorcheckdir":
		// errorcheck all files in lexicographic order
		// useful for finding importing errors
		longdir := filepath.Join(cwd, t.goDirName())
		pkgs, err := goDirPackages(longdir)
		if err != nil {
			t.err = err
			return
		}
		for i, gofiles := range pkgs {
			out, err := compileInDir(runcmd, longdir, gofiles...)
			if i == len(pkgs)-1 {
				if wantError && err == nil {
					t.err = fmt.Errorf("compilation succeeded unexpectedly\n%s", out)
					return
				} else if !wantError && err != nil {
					t.err = err
					return
				}
			} else if err != nil {
				t.err = err
				return
			}
			var fullshort []string
			for _, name := range gofiles {
				fullshort = append(fullshort, filepath.Join(longdir, name), name)
			}
			t.err = t.errorCheck(string(out), fullshort...)
			if t.err != nil {
				break
			}
		}

	case "rundir":
		// Compile all files in the directory in lexicographic order.
		// then link as if the last file is the main package and run it
		longdir := filepath.Join(cwd, t.goDirName())
		pkgs, err := goDirPackages(longdir)
		if err != nil {
			t.err = err
			return
		}
		for i, gofiles := range pkgs {
			_, err := compileInDir(runcmd, longdir, gofiles...)
			if err != nil {
				t.err = err
				return
			}
			if i == len(pkgs)-1 {
				err = linkFile(runcmd, gofiles[0])
				if err != nil {
					t.err = err
					return
				}
				var cmd []string
				cmd = append(cmd, findExecCmd()...)
				cmd = append(cmd, filepath.Join(t.tempDir, "a.exe"))
				cmd = append(cmd, args...)
				out, err := runcmd(cmd...)
				if err != nil {
					t.err = err
					return
				}
				if strings.Replace(string(out), "\r\n", "\n", -1) != t.expectedOutput() {
					t.err = fmt.Errorf("incorrect output\n%s", out)
				}
			}
		}

	case "build":
		_, err := runcmd("go", "build", "-o", "a.exe", long)
		if err != nil {
			t.err = err
		}

	case "run":
		useTmp = false
		// GOPHERJS.
		out, err := runcmd(append([]string{"gopherjs", "run", t.goFileName()}, args...)...)
		if err != nil {
			t.err = err
			return
		}
		if strings.Replace(string(out), "\r\n", "\n", -1) != t.expectedOutput() {
			t.err = fmt.Errorf("incorrect output\n%s", out)
		}

	case "runoutput":
		rungatec <- true
		defer func() {
			<-rungatec
		}()
		useTmp = false
		out, err := runcmd(append([]string{"go", "run", t.goFileName()}, args...)...)
		if err != nil {
			t.err = err
			return
		}
		tfile := filepath.Join(t.tempDir, "tmp__.go")
		if err := os.WriteFile(tfile, out, 0o666); err != nil {
			t.err = fmt.Errorf("write tempfile:%s", err)
			return
		}
		out, err = runcmd("go", "run", tfile)
		if err != nil {
			t.err = err
			return
		}
		if string(out) != t.expectedOutput() {
			t.err = fmt.Errorf("incorrect output\n%s", out)
		}

	case "errorcheckoutput":
		useTmp = false
		out, err := runcmd(append([]string{"go", "run", t.goFileName()}, args...)...)
		if err != nil {
			t.err = err
			return
		}
		tfile := filepath.Join(t.tempDir, "tmp__.go")
		err = os.WriteFile(tfile, out, 0o666)
		if err != nil {
			t.err = fmt.Errorf("write tempfile:%s", err)
			return
		}
		cmdline := []string{"go", "tool", "compile", "-e", "-o", "a.o"}
		cmdline = append(cmdline, flags...)
		cmdline = append(cmdline, tfile)
		out, err = runcmd(cmdline...)
		if wantError {
			if err == nil {
				t.err = fmt.Errorf("compilation succeeded unexpectedly\n%s", out)
				return
			}
		} else {
			if err != nil {
				t.err = err
				return
			}
		}
		t.err = t.errorCheck(string(out), tfile, "tmp__.go")
		return
	}
}

var execCmd []string

func findExecCmd() []string {
	if execCmd != nil {
		return execCmd
	}
	execCmd = []string{} // avoid work the second time
	if goos == runtime.GOOS && goarch == runtime.GOARCH {
		return execCmd
	}
	path, err := exec.LookPath(fmt.Sprintf("go_%s_%s_exec", goos, goarch))
	if err == nil {
		execCmd = []string{path}
	}
	return execCmd
}

func (t *test) String() string {
	return filepath.Join(t.dir, t.gofile)
}

func (t *test) makeTempDir() {
	var err error
	t.tempDir, err = os.MkdirTemp("", "")
	check(err)
}

func (t *test) expectedOutput() string {
	filename := filepath.Join(t.dir, t.gofile)
	filename = filename[:len(filename)-len(".go")]
	filename += ".out"
	b, _ := os.ReadFile(filename)
	return string(b)
}

func splitOutput(out string) []string {
	// gc error messages continue onto additional lines with leading tabs.
	// Split the output at the beginning of each line that doesn't begin with a tab.
	// <autogenerated> lines are impossible to match so those are filtered out.
	var res []string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasSuffix(line, "\r") { // remove '\r', output by compiler on windows
			line = line[:len(line)-1]
		}
		if strings.HasPrefix(line, "\t") {
			res[len(res)-1] += "\n" + line
		} else if strings.HasPrefix(line, "go tool") || strings.HasPrefix(line, "<autogenerated>") {
			continue
		} else if strings.TrimSpace(line) != "" {
			res = append(res, line)
		}
	}
	return res
}

func (t *test) errorCheck(outStr string, fullshort ...string) (err error) {
	defer func() {
		if *verbose && err != nil {
			log.Printf("%s gc output:\n%s", t, outStr)
		}
	}()
	var errs []error
	out := splitOutput(outStr)

	// Cut directory name.
	for i := range out {
		for j := 0; j < len(fullshort); j += 2 {
			full, short := fullshort[j], fullshort[j+1]
			out[i] = strings.Replace(out[i], full, short, -1)
		}
	}

	var want []wantedError
	for j := 0; j < len(fullshort); j += 2 {
		full, short := fullshort[j], fullshort[j+1]
		want = append(want, t.wantedErrors(full, short)...)
	}

	for _, we := range want {
		var errmsgs []string
		errmsgs, out = partitionStrings(we.prefix, out)
		if len(errmsgs) == 0 {
			errs = append(errs, fmt.Errorf("%s:%d: missing error %q", we.file, we.lineNum, we.reStr))
			continue
		}
		matched := false
		n := len(out)
		for _, errmsg := range errmsgs {
			if we.re.MatchString(errmsg) {
				matched = true
			} else {
				out = append(out, errmsg)
			}
		}
		if !matched {
			errs = append(errs, fmt.Errorf("%s:%d: no match for %#q in:\n\t%s", we.file, we.lineNum, we.reStr, strings.Join(out[n:], "\n\t")))
			continue
		}
	}

	if len(out) > 0 {
		errs = append(errs, fmt.Errorf("Unmatched Errors:"))
		for _, errLine := range out {
			errs = append(errs, fmt.Errorf("%s", errLine))
		}
	}

	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "\n")
	for _, err := range errs {
		fmt.Fprintf(&buf, "%s\n", err.Error())
	}
	return errors.New(buf.String())
}

func (t *test) updateErrors(out string, file string) {
	// Read in source file.
	src, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	lines := strings.Split(string(src), "\n")
	// Remove old errors.
	for i, ln := range lines {
		pos := strings.Index(ln, " // ERROR ")
		if pos >= 0 {
			lines[i] = ln[:pos]
		}
	}
	// Parse new errors.
	errors := make(map[int]map[string]bool)
	tmpRe := regexp.MustCompile(`autotmp_[0-9]+`)
	for _, errStr := range splitOutput(out) {
		colon1 := strings.Index(errStr, ":")
		if colon1 < 0 || errStr[:colon1] != file {
			continue
		}
		colon2 := strings.Index(errStr[colon1+1:], ":")
		if colon2 < 0 {
			continue
		}
		colon2 += colon1 + 1
		line, err := strconv.Atoi(errStr[colon1+1 : colon2])
		line--
		if err != nil || line < 0 || line >= len(lines) {
			continue
		}
		msg := errStr[colon2+2:]
		for _, r := range []string{`\`, `*`, `+`, `[`, `]`, `(`, `)`} {
			msg = strings.Replace(msg, r, `\`+r, -1)
		}
		msg = strings.Replace(msg, `"`, `.`, -1)
		msg = tmpRe.ReplaceAllLiteralString(msg, `autotmp_[0-9]+`)
		if errors[line] == nil {
			errors[line] = make(map[string]bool)
		}
		errors[line][msg] = true
	}
	// Add new errors.
	for line, errs := range errors {
		var sorted []string
		for e := range errs {
			sorted = append(sorted, e)
		}
		sort.Strings(sorted)
		lines[line] += " // ERROR"
		for _, e := range sorted {
			lines[line] += fmt.Sprintf(` "%s$"`, e)
		}
	}
	// Write new file.
	err = os.WriteFile(file, []byte(strings.Join(lines, "\n")), 0o640)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	// Polish.
	exec.Command("go", "fmt", file).CombinedOutput()
}

// matchPrefix reports whether s is of the form ^(.*/)?prefix(:|[),
// That is, it needs the file name prefix followed by a : or a [,
// and possibly preceded by a directory name.
func matchPrefix(s, prefix string) bool {
	i := strings.Index(s, ":")
	if i < 0 {
		return false
	}
	j := strings.LastIndex(s[:i], "/")
	s = s[j+1:]
	if len(s) <= len(prefix) || s[:len(prefix)] != prefix {
		return false
	}
	switch s[len(prefix)] {
	case '[', ':':
		return true
	}
	return false
}

func partitionStrings(prefix string, strs []string) (matched, unmatched []string) {
	for _, s := range strs {
		if matchPrefix(s, prefix) {
			matched = append(matched, s)
		} else {
			unmatched = append(unmatched, s)
		}
	}
	return
}

type wantedError struct {
	reStr   string
	re      *regexp.Regexp
	lineNum int
	file    string
	prefix  string
}

var (
	errRx       = regexp.MustCompile(`// (?:GC_)?ERROR (.*)`)
	errQuotesRx = regexp.MustCompile(`"([^"]*)"`)
	lineRx      = regexp.MustCompile(`LINE(([+-])([0-9]+))?`)
)

func (t *test) wantedErrors(file, short string) (errs []wantedError) {
	cache := make(map[string]*regexp.Regexp)

	src, _ := os.ReadFile(file)
	for i, line := range strings.Split(string(src), "\n") {
		lineNum := i + 1
		if strings.Contains(line, "////") {
			// double comment disables ERROR
			continue
		}
		m := errRx.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		all := m[1]
		mm := errQuotesRx.FindAllStringSubmatch(all, -1)
		if mm == nil {
			log.Fatalf("%s:%d: invalid errchk line: %s", t.goFileName(), lineNum, line)
		}
		for _, m := range mm {
			rx := lineRx.ReplaceAllStringFunc(m[1], func(m string) string {
				n := lineNum
				if strings.HasPrefix(m, "LINE+") {
					delta, _ := strconv.Atoi(m[5:])
					n += delta
				} else if strings.HasPrefix(m, "LINE-") {
					delta, _ := strconv.Atoi(m[5:])
					n -= delta
				}
				return fmt.Sprintf("%s:%d", short, n)
			})
			re := cache[rx]
			if re == nil {
				var err error
				re, err = regexp.Compile(rx)
				if err != nil {
					log.Fatalf("%s:%d: invalid regexp \"%s\" in ERROR line: %v", t.goFileName(), lineNum, rx, err)
				}
				cache[rx] = re
			}
			prefix := fmt.Sprintf("%s:%d", short, lineNum)
			errs = append(errs, wantedError{
				reStr:   rx,
				re:      re,
				prefix:  prefix,
				lineNum: lineNum,
				file:    short,
			})
		}
	}

	return
}

// defaultRunOutputLimit returns the number of runoutput tests that
// can be executed in parallel.
func defaultRunOutputLimit() int {
	const maxArmCPU = 2

	cpu := runtime.NumCPU()
	if runtime.GOARCH == "arm" && cpu > maxArmCPU {
		cpu = maxArmCPU
	}
	return cpu
}

// checkShouldTest runs sanity checks on the shouldTest function.
func checkShouldTest() {
	assert := func(ok bool, _ string) {
		if !ok {
			panic("fail")
		}
	}
	assertNot := func(ok bool, _ string) { assert(!ok, "") }

	// Simple tests.
	assert(shouldTest("// +build linux", "linux", "arm"))
	assert(shouldTest("// +build !windows", "linux", "arm"))
	assertNot(shouldTest("// +build !windows", "windows", "amd64"))

	// A file with no build tags will always be tested.
	assert(shouldTest("// This is a test.", "os", "arch"))

	// Build tags separated by a space are OR-ed together.
	assertNot(shouldTest("// +build arm 386", "linux", "amd64"))

	// Build tags separated by a comma are AND-ed together.
	assertNot(shouldTest("// +build !windows,!plan9", "windows", "amd64"))
	assertNot(shouldTest("// +build !windows,!plan9", "plan9", "386"))

	// Build tags on multiple lines are AND-ed together.
	assert(shouldTest("// +build !windows\n// +build amd64", "linux", "amd64"))
	assertNot(shouldTest("// +build !windows\n// +build amd64", "windows", "amd64"))

	// Test that (!a OR !b) matches anything.
	assert(shouldTest("// +build !windows !plan9", "windows", "amd64"))

	// GOPHERJS: Custom rule, test that don't run on nacl should also not run on js.
	assertNot(shouldTest("// +build !nacl,!plan9,!windows", "darwin", "js"))
}

// envForDir returns a copy of the environment
// suitable for running in the given directory.
// The environment is the current process's environment
// but with an updated $PWD, so that an os.Getwd in the
// child will be faster.
func envForDir(dir string) []string {
	env := os.Environ()
	for i, kv := range env {
		if strings.HasPrefix(kv, "PWD=") {
			env[i] = "PWD=" + dir
			return env
		}
	}
	env = append(env, "PWD="+dir)
	return env
}

func getenv(key, def string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return def
}

// splitQuoted splits the string s around each instance of one or more consecutive
// white space characters while taking into account quotes and escaping, and
// returns an array of substrings of s or an empty list if s contains only white space.
// Single quotes and double quotes are recognized to prevent splitting within the
// quoted region, and are removed from the resulting substrings. If a quote in s
// isn't closed err will be set and r will have the unclosed argument as the
// last element. The backslash is used for escaping.
//
// For example, the following string:
//
//	a b:"c d" 'e''f'  "g\""
//
// Would be parsed as:
//
//	[]string{"a", "b:c d", "ef", `g"`}
//
// [copied from src/go/build/build.go]
func splitQuoted(s string) (r []string, err error) {
	var args []string
	arg := make([]rune, len(s))
	escaped := false
	quoted := false
	quote := '\x00'
	i := 0
	for _, rune := range s {
		switch {
		case escaped:
			escaped = false
		case rune == '\\':
			escaped = true
			continue
		case quote != '\x00':
			if rune == quote {
				quote = '\x00'
				continue
			}
		case rune == '"' || rune == '\'':
			quoted = true
			quote = rune
			continue
		case unicode.IsSpace(rune):
			if quoted || i > 0 {
				quoted = false
				args = append(args, string(arg[:i]))
				i = 0
			}
			continue
		}
		arg[i] = rune
		i++
	}
	if quoted || i > 0 {
		args = append(args, string(arg[:i]))
	}
	if quote != 0 {
		err = errors.New("unclosed quote")
	} else if escaped {
		err = errors.New("unfinished escaping")
	}
	return args, err
}
