package gorepo_test

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

// Go repository basic compiler tests, and regression tests for fixed compiler bugs.
func TestGoRepositoryCompilerTests(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Go repository tests in the short mode")
	}
	if runtime.GOOS == "js" {
		t.Skip("test meant to be run using normal Go compiler (needs os/exec)")
	}

	args := []string{"go", "run", "run.go", "-summary"}
	if testing.Verbose() {
		args = append(args, "-v")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
}
