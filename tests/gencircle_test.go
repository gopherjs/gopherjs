package tests_test

import (
	"embed"
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

func Test_GenCircle_CatBox(t *testing.T) {
	// TODO(grantnelson-wf): This test hits an error similar to
	// `panic: info did not have function declaration instance for
	// "collections.Push[box.Unboxer[cat.Cat]]"` from `analysis/Info:IsBlocking`.
	//
	// This is because no instance of `Stack` is used explicitly in code,
	// i.e. the code doesn't have an `ast.Ident` for `Stack` found in `types.Info.Instances`
	// since the only `Stack` identifiers are the ones for the generic declaration.
	// `Stack[box.Unboxer[cat.Cat]]` is implicitly defined via the call
	// `collections.NewStack[box.Unboxer[cat.Cat]]()` in main.go.
	//
	// We need to update the `typeparams.collector` to add these implicit types
	// to the `PackageInstanceSets` so that `analysis/info` has the implicit
	// instances of `Stack`.
	//
	// Simply adding `_ = collections.Stack[box.Unboxer[cat.Cat]]{}` is a
	// work around for `Stack` issue but the code gets tripped up on `boxImp[T]`
	// via `Box[T]` not being defined since again `boxImp` has not been collected.
	t.Skip(`Implicit Instance Not Yet Collected`)
	runGenCircleTest(t, `catbox`)
}

// Cache buster: Keeping the tests from using cached results when only
// the test application files are changed.
//
//go:embed testdata/gencircle
var _ embed.FS

func runGenCircleTest(t *testing.T, testPkg string) {
	t.Helper()
	if runtime.GOOS == `js` {
		t.Skip(`test meant to be run using normal Go compiler (needs os/exec)`)
	}

	const (
		basePath = `testdata/gencircle`
		mainFile = `main.go`
		outFile  = `main.out`
	)

	mainPath := filepath.Join(basePath, testPkg, mainFile)
	gotBytes, err := exec.Command(`gopherjs`, `run`, mainPath).CombinedOutput()
	got := normalizeOut(gotBytes)
	if err != nil {
		t.Fatalf("error from exec: %v:\n%s", err, got)
	}

	outPath := filepath.Join(basePath, testPkg, outFile)
	wantBytes, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf(`error reading .out file: %v`, err)
	}
	want := normalizeOut(wantBytes)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Got diff (-want,+got):\n%s", diff)
	}
}

func normalizeOut(b []byte) string {
	s := string(b)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}
