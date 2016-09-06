// +build dev

package natives

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gopherjs/gopherjs/go/build"
	"github.com/shurcooL/httpfs/filter"
)

func importPathToDir(importPath string) string {
	p, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		log.Fatalln(err)
	}
	return p.Dir
}

// FS is a virtual filesystem that contains native packages.
var FS = filter.Keep(
	http.Dir(importPathToDir("github.com/gopherjs/gopherjs/compiler/natives")),
	func(path string, fi os.FileInfo) bool {
		return path == "/" || path == "/src" || strings.HasPrefix(path, "/src/") ||
			path == "/pkg" || strings.HasPrefix(path, "/pkg/") // TODO: pkg folder is avoidable.
	},
)
