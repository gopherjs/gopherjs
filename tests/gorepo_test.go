package tests_test

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

// Go repository basic compiler tests, and regression tests for fixed compiler bugs.
func TestGoRepositoryCompilerTests(t *testing.T) {
	if runtime.GOARCH == "js" {
		t.Skip("test meant to be run using normal Go compiler (needs os/exec)")
	}

	args := []string{"go", "run", "run.go", "-summary"}
	if testing.Verbose() {
		args = append(args, "-v")
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	cmd.Env = append(os.Environ(), "SOURCE_MAP_SUPPORT=false")
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
}
