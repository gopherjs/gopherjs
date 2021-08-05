// Package versionhack makes sure go/build doesn't disable module support
// whenever GopherJS is compiled by a different Go version than it's targeted
// Go version.
//
// Under the hood, go/build relies on `go list` utility for module support; more
// specifically, for package location discovery. Since ReleaseTags are
// effectively baked into the go binary and can't be overridden, it needs to
// ensure that ReleaseTags set in a go/build.Context instance match the Go tool.
//
// However, it naively assumes that the go tool version in the PATH matches the
// version that was used to build GopherJS and disables module support whenever
// ReleaseTags in the context are set to anything other than the default. This,
// unfortunately, isn't very helpful since gopherjs may be built by a Go version
// other than the PATH's default.
//
// Luckily, even if go tool version is mismatched, it's only used for discovery
// of the package locations, and go/build evaluates build constraints on its own
// with ReleaseTags we've passed.
//
// A better solution would've been for go/build to use go tool from GOROOT and
// check its version against build tags: https://github.com/golang/go/issues/46856.
//
// Until that issue is fixed, we trick go/build into thinking that whatever
// ReleaseTags we've passed are indeed the default. We gain access to the
// variable go/build checks against using "go:linkname" directive and override
// its content as we wish.
package versionhack

import (
	"go/build" // Must be initialized before this package.

	"github.com/gopherjs/gopherjs/compiler"

	_ "unsafe" // For go:linkname
)

//go:linkname releaseTags go/build.defaultReleaseTags
var releaseTags []string

//go:linkname toolTags go/build.defaultToolTags
var toolTags []string

func init() {
	releaseTags = build.Default.ReleaseTags[:compiler.GoVersion]
	toolTags = []string{}
}
