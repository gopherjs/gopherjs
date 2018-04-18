// +build ignore

package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	self = "github.com/gopherjs/gopherjs"
)

var (
	fDebug       = flag.Bool("debug", false, "include debug output")
	fParallelism = flag.Int("p", runtime.NumCPU(), "number of pkgs to test in parallel")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run tests: %v\n", err)
		os.Exit(1)
	}
}

func run() error {

	bpkg, err := build.Import(self, ".", build.FindOnly)
	if err != nil {
		return fmt.Errorf("failed to get directory for import %v: %v", self, err)
	}

	exPath := filepath.Join(bpkg.Dir, "std_test_pkg_exclusions")

	exFi, err := os.Open(exPath)
	if err != nil {
		return fmt.Errorf("error opening %v: %v", exPath, err)
	}

	excls := make(map[string]bool)

	{
		sc := bufio.NewScanner(exFi)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())

			if strings.HasPrefix(line, "#") {
				continue
			}

			excls[line] = true
		}
		if err := sc.Err(); err != nil {
			return fmt.Errorf("failed to line scan %v: %v", exPath, err)
		}
	}

	cmd := exec.Command("go", "list", "std")
	stdListOut, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to %v: %v", strings.Join(cmd.Args, " "), err)
	}

	tests := []string{
		"github.com/gopherjs/gopherjs/tests",
		"github.com/gopherjs/gopherjs/tests/main",
		"github.com/gopherjs/gopherjs/js",
	}

	{
		sc := bufio.NewScanner(strings.NewReader(string(stdListOut)))
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())

			if strings.HasPrefix(line, "#") {
				continue
			}

			if !excls[line] {
				tests = append(tests, line)
			}
		}
		if err := sc.Err(); err != nil {
			return fmt.Errorf("failed to line scan %q: %v", strings.Join(cmd.Args, " "), err)
		}
	}

	var cmds []string

	for _, t := range tests {
		cmds = append(cmds, fmt.Sprintf("gopherjs test -m %v\n", t))
	}

	p := *fParallelism

	debugf("running tests with parallelism %v\n", p)

	testCmd := exec.Command("concsh", "-conc", fmt.Sprintf("%v", p))
	testCmd.Stdin = strings.NewReader(strings.Join(cmds, ""))
	testCmd.Stdout = os.Stdout
	testCmd.Stderr = os.Stderr

	if err := testCmd.Run(); err != nil {
		return fmt.Errorf("test process exited with an error: %v", err)
	}

	return nil
}

func debugf(format string, args ...interface{}) {
	if *fDebug {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}
