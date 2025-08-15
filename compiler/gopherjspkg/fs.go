package gopherjspkg

import "net/http"

// FS is a virtual filesystem that contains core GopherJS packages.
var FS http.FileSystem

// RegisterFS allows setting the embedded fs from another package.
func RegisterFS(fs http.FileSystem) {
	FS = fs
}
