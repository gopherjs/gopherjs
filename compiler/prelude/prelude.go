package prelude

import (
	_ "embed"
	"path/filepath"
	"runtime/debug"
	"strings"
)

//go:embed prelude.js
var prelude string

//go:embed types.js
var types string

//go:embed numeric.js
var numeric string

//go:embed jsmapping.js
var jsmapping string

//go:embed goroutines.js
var goroutines string

type PreludeFile struct {
	Name   string
	Source string
}

// PreludeFiles gets the GopherJS JavaScript interop layers.
func PreludeFiles() (files []PreludeFile) {
	basePath := getPackagePath()
	add := func(name, src string) {
		files = append(files, PreludeFile{
			Name:   filepath.Join(basePath, name),
			Source: src,
		})
	}

	add(`prelude.js`, prelude)
	add(`numberic.js`, numeric)
	add(`types.js`, types)
	add(`goroutines.js`, goroutines)
	add(`jsmapping.js`, jsmapping)
	return
}

// getPackagePath attempts to determine the package path of the prelude package
// by inspecting the runtime stack. This is used to set the correct paths for
// the prelude files so that source maps work correctly. This should get the
// path that the prelude package is locating when GopherJS is compiled.
func getPackagePath() string {
	// Line 0 is goroutine number, line 1 and 2 is runtime/debug.Stack and path,
	// line 3 is this function itself, with line 4 being the path for this function.
	const pathLineIndex = 4
	stack := string(debug.Stack())
	lines := strings.Split(stack, "\n")
	if len(lines) >= pathLineIndex {
		tracePath := strings.TrimSpace(lines[pathLineIndex])
		// tracePath should be in the form: "/full/path/to/compiler/prelude/prelude.go:42 +0x123"
		// so drop the ending to isolate the folder path.
		if index := strings.LastIndex(tracePath, `/`); index >= 0 {
			return tracePath[:index]
		}
	}
	// Fallback to a default path. The source maps may not have the correct path
	// to open the file in an editor but it will be close enough to be useful.
	return `github.com/gopherjs/gopherjs/compiler/prelude/`
}
