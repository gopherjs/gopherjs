package main_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestBasicHashStaleness(t *testing.T) {
	defer func() {
		err := recover()
		if err != nil {
			t.Fatalf("got an expected error: %v", err.(error))
		}
	}()

	h := newHashTester(t)

	td := h.tempDir()
	defer os.RemoveAll(td)
	h.setEnv("GOPATH", td)
	h.dir = h.mkdir(td, "src", "example.com", "rubbish")
	h.mkdir(h.dir, "blah")
	h.writeFile("main.go", `
	package main
	import "example.com/rubbish/blah"
	func main() {
		print(blah.Name)
	}
	`)
	h.writeFile(filepath.Join("blah", "blah.go"), `
	package blah
	const Name = "blah"
	`)
	m := filepath.Join(td, "bin", "rubbish.js")
	a := filepath.Join(td, "pkg", fmt.Sprintf("%v_js", runtime.GOOS), "example.com", "rubbish", "blah.a")

	// variables to hold the current (c) and new (n) archive (a) and main (m)
	// os.FileInfos
	var ca, cm, na, nm os.FileInfo

	// at this point neither main nor archive should exist
	if h.statFile(m) != nil {
		t.Fatalf("main %v existed when it shouldn't have", m)
	}
	if h.statFile(a) != nil {
		t.Fatalf("archive %v existed when it shouldn't have", a)
	}

	h.run("gopherjs", "install", "example.com/rubbish")

	// now both main and the archive should exist
	ca = h.statFile(a)
	if ca == nil {
		t.Fatalf("archive %v should exist but doesn't", a)
	}
	cm = h.statFile(m)
	if cm == nil {
		t.Fatalf("main %v should exist but doesn't", a)
	}

	// re-running the install will cause main to be rewritten; not the package archive
	h.run("gopherjs", "install", "example.com/rubbish")
	nm = h.statFile(m)
	if !nm.ModTime().After(cm.ModTime()) {
		t.Fatalf("expected to see modified main file %v; got %v; prev %v", m, nm.ModTime(), cm.ModTime())
	}
	cm = nm
	if na := h.statFile(a); !na.ModTime().Equal(ca.ModTime()) {
		t.Fatalf("expected not to see modified archive file %v; got %v; want %v", a, na.ModTime(), ca.ModTime())
	}

	// touching the package file should have no effect on the archive
	h.touch(filepath.Join("blah", "blah.go"))
	h.run("gopherjs", "install", "example.com/rubbish/blah") // only install the package here
	if na := h.statFile(a); !na.ModTime().Equal(ca.ModTime()) {
		t.Fatalf("expected not to see modified archive file %v; got %v; want %v", a, na.ModTime(), ca.ModTime())
	}

	// now update package file - should cause modification time change
	h.writeFile(filepath.Join("blah", "blah.go"), `
	package blah
	const Name = "GopherJS"
	`)
	h.run("gopherjs", "install", "example.com/rubbish")
	na = h.statFile(a)
	if !na.ModTime().After(ca.ModTime()) {
		t.Fatalf("expected to see modified archive file %v; got %v; prev %v", a, na.ModTime(), ca.ModTime())
	}
	ca = na

	// now change build tags - should cause modification time change
	h.run("gopherjs", "install", "--tags", "asdf", "example.com/rubbish")
	na = h.statFile(a)
	if !na.ModTime().After(ca.ModTime()) {
		t.Fatalf("expected to see modified archive file %v; got %v; prev %v", a, na.ModTime(), ca.ModTime())
	}
	ca = na
}

type hashTester struct {
	t   *testing.T
	dir string
	env []string
}

func newHashTester(t *testing.T) *hashTester {
	wd, err := os.Getwd()
	if err != nil {
		fatalf("run failed to get working directory: %v", err)
	}
	return &hashTester{
		t:   t,
		dir: wd,
		env: os.Environ(),
	}
}

func (h *hashTester) touch(path string) {
	path = filepath.Join(h.dir, path)
	now := time.Now().UTC()
	if err := os.Chtimes(path, now, now); err != nil {
		fatalf("failed to touch %v: %v", path, err)
	}
}

func (h *hashTester) statFile(path string) os.FileInfo {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		fatalf("failed to stat %v: %v", path, err)
	}

	return fi
}

func (h *hashTester) setEnv(key, val string) {
	newEnv := []string{fmt.Sprintf("%v=%v", key, val)}
	for _, e := range h.env {
		if !strings.HasPrefix(e, key+"=") {
			newEnv = append(newEnv, e)
		}
	}
	h.env = newEnv
}

func (h *hashTester) mkdir(dirs ...string) string {
	d := filepath.Join(dirs...)
	if err := os.MkdirAll(d, 0755); err != nil {
		fatalf("failed to mkdir %v: %v\n", d, err)
	}
	return d
}

func (h *hashTester) writeFile(path, contents string) {
	path = filepath.Join(h.dir, path)
	if err := ioutil.WriteFile(path, []byte(contents), 0644); err != nil {
		fatalf("failed to write file %v: %v", path, err)
	}
}

func (h *hashTester) tempDir() string {
	h.t.Helper()

	td, err := ioutil.TempDir("", "gopherjs_hashTester")
	if err != nil {
		fatalf("failed to create temp dir: %v", err)
	}

	return td
}

func (h *hashTester) run(c string, args ...string) {
	h.t.Helper()

	cmd := exec.Command(c, args...)
	cmd.Dir = h.dir
	cmd.Env = h.env

	out, err := cmd.CombinedOutput()
	if err != nil {
		fullCmd := append([]string{c}, args...)
		fatalf("failed to run %v: %v\n%v", strings.Join(fullCmd, " "), err, string(out))
	}
}

func fatalf(format string, args ...interface{}) {
	panic(fmt.Errorf(format, args...))
}
