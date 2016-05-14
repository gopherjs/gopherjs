// +build dev

package natives

import (
	"go/build"
	"log"
	"net/http"
	"os"
	pathpkg "path"

	"github.com/shurcooL/httpfs/filter"
)

func importPathToDir(importPath string) string {
	p, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		log.Fatalln(err)
	}
	return p.Dir
}

// Data is a virtual filesystem that contains native packages.
var Data = filter.New(
	http.Dir(importPathToDir("github.com/gopherjs/gopherjs/compiler/natives")),
	func(path string, fi os.FileInfo) bool {
		if pathpkg.Dir(path) == "/" && !fi.IsDir() {
			// Skip all files (not directories) in root folder.
			return true
		}
		return false
	},
)
