package tests_test

import (
	"embed"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_GenCircle_Simple(t *testing.T) { runGenCircleTest(t, `simple`) }

func Test_GenCircle_PingPong(t *testing.T) { runGenCircleTest(t, `pingpong`) }

func Test_GenCircle_Burninate(t *testing.T) { runGenCircleTest(t, `burninate`) }

func Test_GenCircle_CatBox(t *testing.T) { runGenCircleTest(t, `catbox`) }

func Test_GenCircle_Trammel(t *testing.T) { runGenCircleTest(t, `trammel`) }

// Cache buster: Keeping the tests from using cached results when only
// the test application files are changed.
//
//go:embed testdata/gencircle
var _ embed.FS

func runGenCircleTest(t *testing.T, testPkg string) {
	t.Helper()
	const basePath = `testdata/gencircle`
	runOutputTest(t, basePath, testPkg)
}

func runOutputTest(t *testing.T, basePath, testPkg string, extraArgs ...string) {
	t.Helper()
	if runtime.GOOS == `js` {
		t.Skip(`test meant to be run using normal Go compiler (needs os/exec)`)
	}

	const (
		mainFile = `main.go`
		outFile  = `main.out`
	)

	mainPath := filepath.Join(basePath, testPkg, mainFile)
	args := append([]string{`run`, mainPath}, extraArgs...)
	gotBytes, err := exec.Command(`gopherjs`, args...).CombinedOutput()
	if err != nil && !errors.Is(err, &exec.ExitError{}) {
		t.Fatalf("unexpected error from exec: %v:\n%s", err, string(gotBytes))
	}

	outPath := filepath.Join(basePath, testPkg, outFile)
	wantBytes, err := os.ReadFile(outPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf(`error reading %s file: %v`, outFile, err)
		}

		// Has no output file so check if output was empty
		if len(gotBytes) > 0 {
			t.Errorf("Expected no output but got:\n%s", string(gotBytes))
		}
		return
	}

	got := normalizeOut(gotBytes)
	want := normalizeOut(wantBytes)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Got diff (-want,+got):\n%s", diff)
	}
}

func normalizeOut(b []byte) []string {
	s := string(b)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}
