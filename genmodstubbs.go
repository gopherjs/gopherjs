// +build ignore

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/rogpeppe/go-internal/module"
)

func main() {
	flag.Parse()

	gmt := exec.Command("go", "mod", "tidy")
	if err := gmt.Run(); err != nil {
		log.Fatalf("failed to run %v: %v\n%s", strings.Join(gmt.Args, " "), err, errToBytes(err))
	}

	gme := exec.Command("go", "mod", "edit", "-json")
	gme.Stderr = new(bytes.Buffer)

	out, err := gme.Output()
	if err != nil {
		log.Fatalf("failed to run %v: %v\n%s", strings.Join(gme.Args, " "), err, errToBytes(err))
	}

	type mod struct {
		Path    string
		Version string
	}

	var goMod struct {
		Require []mod
	}

	if err := json.Unmarshal(out, &goMod); err != nil {
		log.Fatalf("failed to unmarshal: %v\n%s", err, out)
	}

	// add in gopherjs v0.0.0 to satisfy our redirect
	goMod.Require = append(goMod.Require, mod{
		Path:    "github.com/gopherjs/gopherjs",
		Version: "v0.0.0",
	})

	dirCmd := exec.Command("go", "list", "-m", "-f={{.Dir}}")
	out, err = dirCmd.Output()
	if err != nil {
		log.Fatalf("failed to run %v: %v\n%s", strings.Join(dirCmd.Args, " "), err, errToBytes(err))
	}

	dir := strings.TrimSpace(string(out))
	mods := filepath.Join(dir, "testdata", "mod")

	curr, err := filepath.Glob(filepath.Join(mods, "*"))
	if err != nil {
		log.Fatalf("failed to glob current mod files: %v", err)
	}

	for _, c := range curr {
		base := filepath.Base(c)
		if strings.HasPrefix(base, "example.com") {
			continue
		}

		if err := os.Remove(c); err != nil {
			log.Fatalf("failed to remove %v: %v", c, err)
		}
	}

	if err := os.MkdirAll(mods, 0755); err != nil {
		log.Fatalf("failed to create directory %v: %v", mods, err)
	}

	tmpl := template.Must(template.New("tmpl").Parse(`
module {{.Path}}@{{.Version}}

-- .mod --
module "{{.Path}}"

-- .info --
{"Version":"{{.Version}}","Time":"2018-05-06T08:24:08Z"}
`[1:]))

	for _, r := range goMod.Require {
		enc, err := module.EncodePath(r.Path)
		if err != nil {
			log.Fatalf("failed to encode path %q: %v", r.Path, err)
		}
		ver, err := module.EncodeVersion(r.Version)
		if err != nil {
			log.Fatalf("failed to encode version %q: %v", r.Version, err)
		}

		prefix := strings.Replace(enc, "/", "_", -1)
		name := filepath.Join(mods, prefix+"_"+ver+".txt")
		f, err := os.Create(name)
		if err != nil {
			log.Fatalf("failed to create %v: %v", name, err)
		}

		if err := tmpl.Execute(f, r); err != nil {
			log.Fatalf("failed to execute template for %v: %v", name, err)
		}

		if err := f.Close(); err != nil {
			log.Fatalf("failed to close %v: %v", name, err)
		}
	}
}

func errToBytes(err error) []byte {
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.Stderr
	}

	return nil
}
