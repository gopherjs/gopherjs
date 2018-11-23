package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"testing"

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
	p := testscript.Params{
		Dir: "testdata",
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"modified": buildModified(),
			"changed":  buildChanged(),
		},
		Setup: func(e *testscript.Env) error {
			e.Vars = append(e.Vars,
				"NODE_PATH="+os.Getenv("NODE_PATH"),
				"GOPROXY="+proxyURL,
			)
			return nil
		},
	}
	if err := gotooltest.Setup(&p); err != nil {
		t.Fatal(err)
	}
	testscript.Run(t, p)
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
	cache := make(map[string]os.FileInfo)

	return func(ts *testscript.TestScript, neg bool, args []string) {
		if len(args) != 1 {
			ts.Fatalf("modified take a single file path argument")
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
	cache := make(map[string][]byte)

	return func(ts *testscript.TestScript, neg bool, args []string) {
		if len(args) != 1 {
			ts.Fatalf("changed take a single file path argument")
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
