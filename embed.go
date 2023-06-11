// This file exists to embed the js/ and nosync/ directories. The "embed"
// package only supports embedding files stored beneath the directory level
// of the go:embed directive, so we do the embedding here, then inject the
// fs reference to the gopherjspkg package, where it's used.  In the future we
// may wish to refactor that, as the gopherjspkg package may not really be
// necessary at all any more.

package main

import (
	"embed"
	"net/http"

	"github.com/gopherjs/gopherjs/compiler/gopherjspkg"
)

//go:embed js nosync
var fs embed.FS

func init() {
	gopherjspkg.RegisterFS(http.FS(fs))
}
