// Package build provides high-level API for building GopherJS targets.
//
// This package acts as a bridge between GopherJS compiler (represented by compiler package) and the
// environment and is responsible for such tasks as package loading, dependency resolution, etc.
//
// Compared to v1, this package now delegates most of the work to go/packages provided by the Go
// Team, which hides most of logic related to the build system (e.g. GOPATH-based, Go modules, or
// other build systems).
package build

import (
	"fmt"

	build_v1 "github.com/gopherjs/gopherjs/build"
	"github.com/gopherjs/gopherjs/compiler"
)

type Options = build_v1.Options

// Session manages build process.
type Session struct {
	opts Options
}

// NewSession initializes a fresh build session.
func NewSession(opts Options) (*Session, error) {
	if opts.Watch {
		return nil, fmt.Errorf("not implemented: build_v2 package doesn't support watch option yet")
	}
	if opts.CreateMapFile || opts.MapToLocalDisk {
		return nil, fmt.Errorf("not implemented: build_v2 package doesn't support source maps yet")
	}
	if opts.Color {
		return nil, fmt.Errorf("not implemented: build_v2 package doesn't support colored output yet")
	}

	return &Session{opts: opts}, nil
}

func (s *Session) Build(patterns ...string) (*compiler.Archive, error) {
	return nil, fmt.Errorf("building is not implemented yet")
}
