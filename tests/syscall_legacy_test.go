package tests

import (
	"os/exec"
	"runtime"
	"testing"
)

// TestLegacySyscall tests raw syscall invocation using node_syscall extension.
//
// This mode is largely deprecated (e.g. we build standard library with GOOS=js),
// but we support using the extension when "legacy_syscall" build tag is set.
// This test can be removed after we stop supporting node_syscall extension.
func TestLegacySyscall(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("This test is supported only under Linux")
	}
	cmd := exec.Command("gopherjs", "run", "--tags=legacy_syscall", "./testdata/legacy_syscall/main.go")
	out, err := cmd.CombinedOutput()
	got := string(out)
	if err != nil {
		t.Log(got)
		t.Fatalf("Failed to run test code under gopherjs: %s", err)
	}
	if want := "Hello, world!\n"; got != want {
		t.Errorf("Got wrong output: %q. Want: %q.", got, want)
	}
}
