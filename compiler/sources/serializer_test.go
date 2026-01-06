package sources

import (
	"bytes"
	"encoding/gob"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/gopherjs/gopherjs/compiler/incjs"
	"github.com/gopherjs/gopherjs/internal/srctesting"
)

func TestRoundTrip(t *testing.T) {
	src1 := `package main
		import "fmt"

		func main() {
			fmt.Println(sayHello("World"))
		}`
	src2 := `package main
		import "fmt"
		// sayHello returns a greeting for the named person.
		func sayHello(name string) string {
			return fmt.Sprintf("Hello, %s!", name)
		}`
	src3 := `alert("Hello, JS!");`

	pkgs := srctesting.ParseSources(t,
		[]srctesting.Source{
			{Name: `main.go`, Contents: []byte(src1)},
			{Name: `util.go`, Contents: []byte(src2)},
		}, nil)
	jsFiles := []incjs.File{
		{
			Path:    `hello.inc.js`,
			ModTime: time.Date(1997, time.August, 29, 6, 14, 0, 0, time.UTC),
			Content: []byte(src3),
		},
	}
	srcs0 := &Sources{
		ImportPath: `main`,
		Dir:        `./`,
		Files:      pkgs.Syntax,
		FileSet:    pkgs.Fset,
		JSFiles:    jsFiles,
	}

	// Serialize sources
	buf := &bytes.Buffer{}
	if err := srcs0.Write(gob.NewEncoder(buf).Encode); err != nil {
		t.Fatalf("failed to serialize sources: %v", err)
	}

	// Deserialize sources
	srcs1 := &Sources{}
	if err := srcs1.Read(gob.NewDecoder(buf).Decode); err != nil {
		t.Fatalf("failed to deserialize sources: %v", err)
	}
	checkSourcesAreEqual(t, srcs0, srcs1)
}

func checkSourcesAreEqual(t *testing.T, orig, other *Sources) {
	t.Helper()
	if orig == nil {
		if other == nil {
			return
		}
		t.Errorf(`expected nil sources`)
		return
	}
	if other == nil {
		t.Errorf(`expected non-nil other source`)
		return
	}

	if other.ImportPath != orig.ImportPath {
		t.Errorf("expected import path %q; got %q", orig.ImportPath, other.ImportPath)
	}
	if other.Dir != orig.Dir {
		t.Errorf("expected dir %q; got %q", orig.Dir, other.Dir)
	}

	if len(other.Files) != len(orig.Files) {
		t.Fatalf("expected %d Go files; got %d files", len(orig.Files), len(other.Files))
	}
	for i, origFile := range orig.Files {
		otherFile := other.Files[i]

		origPos := orig.FileSet.Position(origFile.Pos())
		otherPos := other.FileSet.Position(otherFile.Pos())
		if origPos != otherPos {
			t.Errorf("file %d: expected pos position %q; got %q", i, origPos, otherPos)
		}

		origEnd := orig.FileSet.Position(origFile.End())
		otherEnd := other.FileSet.Position(otherFile.End())
		if origEnd != otherEnd {
			t.Errorf("file %d: expected end position %q; got %q", i, origEnd, otherEnd)
		}

		origSrc := srctesting.Format(t, orig.FileSet, origFile)
		otherSrc := srctesting.Format(t, other.FileSet, otherFile)
		if diff := cmp.Diff(origSrc, otherSrc); len(diff) > 0 {
			t.Errorf("the other Go file #%d produces a different formatted output:\n%s", i, diff)
		}
	}

	if len(other.JSFiles) != len(orig.JSFiles) {
		t.Fatalf("expected %d JS files; got %d JS files", len(orig.JSFiles), len(other.JSFiles))
	}
	for i, origJS := range orig.JSFiles {
		otherJS := other.JSFiles[i]

		if origJS.Path != otherJS.Path {
			t.Errorf("JS file %d: expected path %q; got %q", i, origJS.Path, otherJS.Path)
		}
		if !origJS.ModTime.Equal(otherJS.ModTime) {
			t.Errorf("JS file %d: expected mod time %v; got %v", i, origJS.ModTime, otherJS.ModTime)
		}
		if diff := cmp.Diff(string(origJS.Content), string(otherJS.Content)); len(diff) > 0 {
			t.Errorf("the other JS file #%d has a different content:\n%s", i, diff)
		}
	}
}
