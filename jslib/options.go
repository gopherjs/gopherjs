package jslib

import (
	"go/build"
	"io"
)

type Options struct {
	GOROOT        string
	GOPATH        string
	Target        io.Writer // here the js is written to
	SourceMap     io.Writer // here the source map is written to (optional)
	Verbose       bool
	Watch         bool
	CreateMapFile bool
}

func (o *Options) Normalize() {
	if o.GOROOT == "" {
		o.GOROOT = build.Default.GOROOT
	}

	if o.GOPATH == "" {
		o.GOPATH = build.Default.GOPATH
	}

	o.Verbose = o.Verbose || o.Watch
}
