// +build gopherjsdev

package natives

import (
	"go/build"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/shurcooL/httpfs/filter"
)

// FS is a virtual filesystem that contains native packages.
var FS = filter.Keep(
	http.Dir(importPathToDir("github.com/gopherjs/gopherjs/compiler/natives")),
	func(path string, fi os.FileInfo) bool {
		return path == "/" || path == "/src" || strings.HasPrefix(path, "/src/")
	},
)

func importPathToDir(importPath string) string {
	for _, src := range build.Default.SrcDirs() {
		dir := filepath.Join(src, importPath)
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	p, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		log.Fatalln(err)
	}
	return p.Dir
}
