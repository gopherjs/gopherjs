package jslib

import (
	"fmt"
	"github.com/gopherjs/gopherjs/build"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Options is the subset of build.Options, that is exposed to the user of jslib
// and is totally optional.
// The creation of a sourcemap is triggered here if SourceMap is not <nil>.
type Options struct {
	GOROOT    string    // defaults to build.Default.GOROOT
	GOPATH    string    // defaults to build.Default.GOPATH
	SourceMap io.Writer // here the source map is written to
}

// toBuildOptions converts to the real build options in build
func (o *Options) toBuildOptions(target io.Writer) *build.Options {
	b := &build.Options{}
	b.Target = target
	if o != nil {
		b.GOROOT = o.GOROOT
		b.GOPATH = o.GOPATH
		b.SourceMap = o.SourceMap
	}
	b.Normalize()
	return b
}

// BuildDir builds a package based on a directory and writes the result to target
// packagePath is the import path relative to options.GOPATH/src
// target must not be nil
// options may be nil (defaults)
func BuildDir(packagePath string, target io.Writer, options *Options) error {
	if target == nil {
		return fmt.Errorf("no target writer given")
	}
	bo := options.toBuildOptions(target)
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

// BuildFile builds a package based on a single file and writes the result to target
// target must not be nil
// options may be nil (defaults)
func BuildFile(r io.Reader, target io.Writer, options *Options) error {
	pb, err := NewBuilder(target, options)
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
// target must not be nil
// options may be nil (defaults)
func NewBuilder(target io.Writer, options *Options) (*packageBuilder, error) {
	if target == nil {
		return nil, fmt.Errorf("no target writer given")
	}
	return &packageBuilder{
		files:   map[string][]byte{},
		options: options.toBuildOptions(target),
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
