package jslib

import (
	"fmt"
	"github.com/gopherjs/gopherjs/build"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// some kind of duplicate of build.Options, but we don't want to bother the user
// with including build just to set the options and we can't use the options here
// from build, because it would trigger an import loop.
// TODO Maybe find a better solution
// The nice thing about this solution is, that we can strip out some options, that
// are not relevant for library usage, i.e. Verbose, Watch and CreateMapFile.
// The creation of a sourcemap is triggered here if SourceMap is not <nil> which is
// a nicer API than in build.Options
type Options struct {
	GOROOT    string    // defaults to build.Default.GOROOT
	GOPATH    string    // defaults to build.Default.GOPATH
	Target    io.Writer // here the js is written to (mandatory)
	SourceMap io.Writer // here the source map is written to (optional)
}

// toBuildOptions converts to the real build options in build
func (o *Options) toBuildOptions() *build.Options {
	b := &build.Options{}
	b.GOROOT = o.GOROOT
	b.GOPATH = o.GOPATH
	b.Target = o.Target
	b.SourceMap = o.SourceMap
	b.Normalize()
	return b
}

// BuildDir builds a package to Options.Target based on a directory
// packagePath is the import path relative to options.GOPATH/src
func BuildDir(packagePath string, options *Options) error {
	if options.Target == nil {
		return fmt.Errorf("no target writer given")
	}
	bo := options.toBuildOptions()
	return buildDir(filepath.Join(bo.GOPATH, "src", packagePath), bo)
}

// buildDir builds a package to Options.Target based on a directory
// packageDir is the complete path
func buildDir(packageDir string, bo *build.Options) error {
	s := build.NewSession(bo)
	return s.BuildDir(packageDir, "main", "")
}

type packageBuilder struct {
	files      map[string][]byte
	options    *build.Options
	packageDir string
}

// BuildFile builds a package to Option.Target based on a single file
func BuildFile(r io.Reader, options *Options) error {
	pb, err := NewBuilder(options)
	if err != nil {
		return err
	}
	err = pb.AddFile("main.go", r)
	if err != nil {
		return err
	}
	return pb.Build()
}

// NewBuilder creates a new package builder. use it to add several files to
// a package before building it
func NewBuilder(options *Options) (*packageBuilder, error) {
	if options.Target == nil {
		return nil, fmt.Errorf("no target writer given")
	}
	return &packageBuilder{
		files:   map[string][]byte{},
		options: options.toBuildOptions(),
	}, nil
}

// AddFile adds a file with the given filename and the content of r to the packageBuilder
func (b *packageBuilder) AddFile(filename string, r io.Reader) error {
	bt, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	b.files[filename] = bt
	return nil
}

// Build creates a build and writes it to Options.Target
func (b *packageBuilder) Build() error {
	err := b.mkTempPkg()
	defer os.RemoveAll(b.packageDir)

	if err != nil {
		return err
	}

	return buildDir(b.packageDir, b.options)
}

// mkTempPkg creates a physical package for the given files
func (b *packageBuilder) mkTempPkg() error {
	dir, err := ioutil.TempDir(filepath.Join(b.options.GOPATH, "src"), "gopherjs_build_tmp_")
	if err != nil {
		return err
	}
	b.packageDir = dir

	for file, data := range b.files {
		err = ioutil.WriteFile(filepath.Join(b.packageDir, file), data, 0666)
		if err != nil {
			return err
		}
	}
	return nil
}
