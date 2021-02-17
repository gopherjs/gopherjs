// +build gopherjsdev

package natives

import (
	"fmt"
	"go/build"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/shurcooL/httpfs/filter"
)

// FS is a virtual filesystem that contains native packages.
var FS = filter.Keep(
	http.Dir(importPathToDir("github.com/goplusjs/gopherjs/compiler/natives")),
	func(path string, fi os.FileInfo) bool {
		return path == "/" || path == "/src" || strings.HasPrefix(path, "/src/")
	},
)

func importPathToDir(importPath string) string {
	p, err := build.Import(importPath, "", build.FindOnly)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("gopherjsdev importpath:", p.Dir)
	return p.Dir
}
