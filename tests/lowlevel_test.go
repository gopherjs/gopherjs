package tests_test

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Test for internalization/externalization of time.Time/Date when time package is imported
// but time.Time is unused, causing it to be DCEed (or time package not imported at all).
//
// See https://github.com/gopherjs/gopherjs/issues/279.
func TestTimeInternalizationExternalization(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("test meant to be run using normal Go compiler (needs os/exec)")
	}

	gotb, err := exec.Command("gopherjs", "run", filepath.Join("testdata", "time_inexternalization.go")).Output()
	got := string(gotb)
	if err != nil {
		t.Fatalf("%v:\n%s", err, got)
	}

	wantb, err := ioutil.ReadFile(filepath.Join("testdata", "time_inexternalization.out"))
	want := string(wantb)
	if err != nil {
		t.Fatalf("error reading .out file: %v", err)
	}
	got = strings.ReplaceAll(got, "\r\n", "\n")
	want = strings.ReplaceAll(want, "\r\n", "\n")

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Got diff (-want,+got):\n%s", diff)
	}
}

func TestDeferBuiltin(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("test meant to be run using normal Go compiler (needs os/exec)")
	}

	got, err := exec.Command("gopherjs", "run", filepath.Join("testdata", "defer_builtin.go")).CombinedOutput()
	if err != nil {
		t.Fatalf("%v:\n%s", err, got)
	}
}
