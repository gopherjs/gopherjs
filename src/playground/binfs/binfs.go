package main

import (
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	dirs := make(map[string][]os.FileInfo)
	files := make(map[string]string)
	readDir := func(dir string) {
		err := filepath.Walk(dir+"/pkg/darwin_amd64_js/", func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				var err error
				dirs[path[len(dir):]], err = ioutil.ReadDir(path)
				return err
			}

			in, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			files[path[len(dir):]] = string(in)
			return nil
		})
		if err != nil {
			panic(err)
		}
	}

	readDir(build.Default.GOROOT)
	readDir(build.Default.GOPATH)

	out, err := os.Create("src/playground/files.go")
	if err != nil {
		panic(err)
	}
	out.WriteString("package main\n\nimport \"os\"\n\nvar dirs = map[string][]os.FileInfo {\n")
	for name, content := range dirs {
		entries := make([]string, len(content))
		for i, entry := range content {
			entries[i] = fmt.Sprintf(`&FileEntry{ name: "%s", mode: %d }`, entry.Name(), entry.Mode())
		}
		fmt.Fprintf(out, "\t\"%s\": []os.FileInfo{ %s },\n", name, strings.Join(entries, ", "))
	}
	out.WriteString("}\n\nvar files = map[string]string {\n")
	for name, content := range files {
		fmt.Fprintf(out, "\t\"%s\": %#v,\n", name, content)
	}
	out.WriteString("}\n")
	out.Close()
}
