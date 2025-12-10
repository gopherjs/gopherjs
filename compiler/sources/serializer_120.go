//go:build go1.20

package sources

import (
	"fmt"
	"go/ast"
	"go/token"
)

func init() {
	finishUnpackingFile = finishUnpackingFile120
}

// finishUnpackingFile120 performs additional processing needed
// after unpacking a file during deserialization for Go 1.20+.
//
// Specifically, it sets the FileStart and FileEnd fields
// that were added in Go 1.20 to ensure correct position calculations
// used by the type checker (when checking file versions).
// For some reason the gob encoding/decoding isn't automatically setting this.
//
// TODO(grantnelson-wf): Determine why the gob encoding/decoding isn't doing this automatically.
func finishUnpackingFile120(f *ast.File, fs *token.FileSet) {
	fp := fs.File(f.Pos())
	if fp == nil {
		panic(fmt.Errorf(`failed to find token.File for ast.File (in %s at %d) during unpacking`, f.Name.Name, f.Pos()))
	}

	if !f.FileStart.IsValid() {
		f.FileStart = token.Pos(fp.Base())
	}
	if !f.FileEnd.IsValid() {
		f.FileEnd = token.Pos(fp.Base() + fp.Size())
	}
}
