package tests_test

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

// Test that GopherJS can be vendored into a project, and then used to build Go programs.
// See issue https://github.com/gopherjs/gopherjs/issues/415.
func TestGopherJSCanBeVendored(t *testing.T) {
	if runtime.GOARCH == "js" {
		t.Skip("test meant to be run using normal Go compiler (needs os/exec)")
	}

	cmd := exec.Command("sh", "gopherjsvendored_test.sh")
	cmd.Stderr = os.Stdout
	got, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if want := "hello using js pkg\n"; string(got) != want {
		t.Errorf("unexpected stdout from gopherjsvendored_test.sh:\ngot:\n%s\nwant:\n%s", got, want)
	}
}
