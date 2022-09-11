//go:build generate
// +build generate

package gopherjspkg

import (
	_ "github.com/shurcooL/vfsgen" // Force go.mod to require this package.
)

//go:generate vfsgendev -source="github.com/gopherjs/gopherjs/compiler/gopherjspkg".FS -tag=gopherjsdev
//go:generate gofmt -w -s fs_vfsdata.go
