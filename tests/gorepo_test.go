// +build !js

package tests_test

import (
	"os"
	"os/exec"
	"strconv"
	"testing"
)

// Go repository basic compiler tests, and regression tests for fixed compiler bugs.
func TestGoRepositoryCompilerTests(t *testing.T) {
	if b, _ := strconv.ParseBool(os.Getenv("SOURCE_MAP_SUPPORT")); os.Getenv("SOURCE_MAP_SUPPORT") != "" && !b {
		t.Fatal("Source maps disabled, but required for this test. Use SOURCE_MAP_SUPPORT=true or unset it completely.")
	}
	if err := exec.Command("node", "--require", "source-map-support/register", "--eval", "").Run(); err != nil {
		t.Fatal("Source maps disabled, but required for this test. Use Node.js 4.x with source-map-support module for nice stack traces.")
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
