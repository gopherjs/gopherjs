// +build ignore

package main

import (
	"fmt"
	"go/build"
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
	bpkg, err := build.Import("github.com/gopherjs/gopherjs", ".", build.FindOnly)
	if err != nil {
		return fmt.Errorf("failed to locate path for github.com/gopherjs/gopherjs/compiler/prelude: %v", err)
	}

	preludeDir := filepath.Join(bpkg.Dir, "compiler", "prelude")

	args := []string{
		filepath.Join(bpkg.Dir, "node_modules", ".bin", "prettier"),
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
