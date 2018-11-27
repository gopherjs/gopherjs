package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rogpeppe/go-internal/goproxytest"
	"github.com/rogpeppe/go-internal/gotooltest"
	"github.com/rogpeppe/go-internal/testscript"
)

var (
	proxyURL string
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(gobinMain{m}, map[string]func() int{
		"gopherjs": main1,
	}))
}

type gobinMain struct {
	m *testing.M
}

func (m gobinMain) Run() int {
	// Start the Go proxy server running for all tests.
	srv, err := goproxytest.NewServer("testdata/mod", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot start proxy: %v", err)
		return 1
	}
	proxyURL = srv.URL

	return m.m.Run()
}

func TestScripts(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	p := testscript.Params{
		Dir: "testdata",
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"modified": buildModified(),
			"changed":  buildChanged(),
			"cpr":      cmdCpr,
			"highport": highport,
			"httpget":  httpget,
			"sleep":    sleep,
		},
		Setup: func(e *testscript.Env) error {
			e.Vars = append(e.Vars,
				"NODE_PATH="+os.Getenv("NODE_PATH"),
				"GOPROXY="+proxyURL,
				"SELF="+wd,
			)
			return nil
		},
	}
	if err := gotooltest.Setup(&p); err != nil {
		t.Fatal(err)
	}
	testscript.Run(t, p)
}

// sleep for the specified duration
func sleep(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("sleep does not understand negation")
	}

	if len(args) != 1 {
		ts.Fatalf("Usage: sleep duration")
	}

	d, err := time.ParseDuration(args[0])
	if err != nil {
		ts.Fatalf("failed to parse duration %q: %v", args[0], err)
	}

	time.Sleep(d)
}

// Usage:
//
//		httpget url outputFile
//
func httpget(ts *testscript.TestScript, neg bool, args []string) {
	if len(args) != 2 {
		ts.Fatalf("Usage: httpget url outputFile")
	}

	url := args[0]
	ofPath := ts.MkAbs(args[1])

	resp, err := http.Get(url)
	if err != nil {
		ts.Fatalf("httpget %v failed: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if !neg {
			ts.Fatalf("httpget %v return status code %v", url, resp.StatusCode)
		}
	}

	of, err := os.Create(ofPath)
	if err != nil {
		ts.Fatalf("failed to create output file %v: %v", ofPath, err)
	}

	if _, err := io.Copy(of, resp.Body); err != nil {
		ts.Fatalf("failed to write response to output file %v: %v", ofPath, err)
	}

	if err := of.Close(); err != nil {
		ts.Fatalf("failed to close output file %v: %v", ofPath, err)
	}
}

// Sets the environment variable named by the single argument key to an
// available high port.
func highport(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("highport does not understand negation")
	}

	if len(args) != 1 {
		ts.Fatalf("highport takes exactly one argument; the name of the environment variable key to set")
	}
	key := args[0]

	l, err := net.Listen("tcp", ":0")
	if err != nil {
		ts.Fatalf("could not get a free high port: %v", err)
	}
	defer func() {
		if err := l.Close(); err != nil {
			ts.Fatalf("failed to free up high port: %v", err)
		}
	}()
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		ts.Fatalf("could not extract port from %q: %v", l.Addr().String(), err)
	}

	ts.Setenv(key, port)
}

// buildModified returns a new instance of a testscript command that determines
// whether the single file argument has been modified since the command was
// last called on that file. Strictly speaking it is only safe to test files
// beneath $WORK because this represents the truly isolated area of a
// testscript run. This is an inherently racey operation in the presence of
// background tasks, hence we don't worry about synchronising.
//
// The first time of calling modified for any given file path is defined to
// return 0, i.e. false
func buildModified() func(ts *testscript.TestScript, neg bool, args []string) {
	var lock sync.Mutex
	cache := make(map[string]os.FileInfo)

	return func(ts *testscript.TestScript, neg bool, args []string) {
		lock.Lock()
		defer lock.Unlock()

		var poorUsage bool
		switch len(args) {
		case 2:
			if args[0] != "-clear" {
				poorUsage = true
			}
		case 1:
			if args[0] == "-clear" {
				poorUsage = true
			}
		default:
			poorUsage = true
		}
		if poorUsage {
			ts.Fatalf("usage: modified [-clear] file")
		}

		if args[0] == "-clear" {
			delete(cache, args[1])
			return
		}

		fp := ts.MkAbs(args[0])

		nfi, err := os.Stat(fp)
		if err != nil {
			ts.Fatalf("failed to stat %v: %v", fp, err)
		}

		if nfi.IsDir() {
			ts.Fatalf("%v is a directory, not a file", fp)
		}

		fi, ok := cache[fp]
		cache[fp] = nfi
		if !ok {
			if !neg {
				ts.Fatalf("%v has not been modified; first time of checking", fp)
			}
			return
		}

		switch {
		case nfi.ModTime().Before(fi.ModTime()):
			ts.Fatalf("file %v now has an earlier modification time (%v -> %v)", fp, fi.ModTime(), nfi.ModTime())
		case nfi.ModTime().Equal(fi.ModTime()):
			if neg {
				ts.Fatalf("%v has not been modified", fp)
			}
			return
		default:
			if neg {
				ts.Fatalf("%v has been modified (%v -> %v)", fp, fi.ModTime(), nfi.ModTime())
			}

			cache[fp] = nfi
		}
	}
}

// buildChanged returns a new instance of a testscript command that determines
// whether the single file argument has been changed since the command was
// last called on that file. Strictly speaking it is only safe to test files
// beneath $WORK because this represents the truly isolated area of a
// testscript run. This is an inherently racey operation in the presence of
// background tasks, hence we don't worry about synchronising.
//
// The first time of calling changed for any given file path is defined to
// return 0, i.e. false
func buildChanged() func(ts *testscript.TestScript, neg bool, args []string) {
	var lock sync.Mutex
	cache := make(map[string][]byte)

	return func(ts *testscript.TestScript, neg bool, args []string) {
		lock.Lock()
		defer lock.Unlock()

		var poorUsage bool
		switch len(args) {
		case 2:
			if args[0] != "-clear" {
				poorUsage = true
			}
		case 1:
			if args[0] == "-clear" {
				poorUsage = true
			}
		default:
			poorUsage = true
		}
		if poorUsage {
			ts.Fatalf("usage: changed [-clear] file")
		}

		if args[0] == "-clear" {
			delete(cache, args[1])
			return
		}

		fp := ts.MkAbs(args[0])

		f, err := os.Open(fp)
		if err != nil {
			ts.Fatalf("failed to open %v: %v", fp, err)
		}
		defer f.Close()

		nhash := sha256.New()
		if _, err := io.Copy(nhash, f); err != nil {
			ts.Fatalf("failed to hash %v: %v", fp, err)
		}

		nsum := nhash.Sum(nil)

		sum, ok := cache[fp]
		cache[fp] = nsum
		if !ok {
			if !neg {
				ts.Fatalf("%v has not been changed; first time of checking", fp)
			}
			return
		}

		eq := bytes.Equal(nsum, sum)
		if eq && !neg {
			ts.Fatalf("file %v not changed", fp)
		}
		if !eq && neg {
			ts.Fatalf("file %v changed", fp)
		}
	}
}

// cmdCpr implements a recursive copy of go files. Takes two arguments: source
// directory and target directory. The source directory must exist.  The target
// directory will be created if it does not exist.
func cmdCpr(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("cpr does not understand negation")
	}
	if len(args) != 2 {
		ts.Fatalf("cpr takes two arguments: got %v", len(args))
	}

	src, dst := ts.MkAbs(args[0]), ts.MkAbs(args[1])

	sfi, err := os.Stat(src)
	if err != nil {
		ts.Fatalf("source %v must exist: %v", src, err)
	}
	if !sfi.IsDir() {
		ts.Fatalf("source %v must be a directory", src)
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		ts.Fatalf("error trying to ensure target directory exists")
	}

	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		dstPath := path
		dstPath = strings.TrimPrefix(dstPath, src)
		dstPath = strings.TrimPrefix(dstPath, string(os.PathSeparator))

		// root
		if dstPath == "" {
			return nil
		}

		dstPath = filepath.Join(dst, dstPath)

		name := info.Name()

		if info.IsDir() {
			switch name[0] {
			case '_', '.':
				return filepath.SkipDir
			}
			if err := os.MkdirAll(dstPath, info.Mode()); err != nil {
				return fmt.Errorf("failed to mkdir %v: %v", dstPath, err)
			}
			return nil
		}

		if !strings.HasSuffix(name, ".go") || name[0] == '_' || name[0] == '.' {
			return nil
		}

		srcf, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open source file %v: %v", path, err)
		}
		dstf, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE, info.Mode())
		if err != nil {
			return fmt.Errorf("failed to create target file %v: %v", dstPath, err)
		}
		if _, err := io.Copy(dstf, srcf); err != nil {
			return fmt.Errorf("failed to copy from %v to %v: %v", path, dstPath, err)
		}

		return nil
	})

	if err != nil {
		ts.Fatalf("failed to recursively copy: %v", err)
	}
}
