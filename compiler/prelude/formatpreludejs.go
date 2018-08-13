// +build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

func run() error {
	var bpkgDir string
	{
		cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
		out, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to resolve module path: %v", err)
		}
		bpkgDir = strings.TrimSpace(string(out))
	}

	preludeDir := filepath.Join(bpkgDir, "compiler", "prelude")

	args := []string{
		filepath.Join(bpkgDir, "node_modules", ".bin", "prettier"),
		"--config",
		filepath.Join(preludeDir, "prettier_options.json"),
		"--write",
	}

	fis, err := ioutil.ReadDir(preludeDir)
	if err != nil {
		return fmt.Errorf("failed to list contents of %v: %v", preludeDir, err)
	}
	for _, fi := range fis {
		fn := fi.Name()
		if !strings.HasSuffix(fn, ".js") || strings.HasSuffix(fn, ".min.js") {
			continue
		}
		args = append(args, fn)
	}

	cmd := exec.Command(args[0], args[1:]...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run %v: %v\n%s", strings.Join(args, " "), err, string(out))
	}

	return nil
}
