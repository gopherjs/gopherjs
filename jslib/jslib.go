package jslib

import (
	"github.com/metakeule/gopherjs/build"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

// BuildDir builds a package to Options.Target based on a directory
// packagePath is the import path relative to options.GOPATH/src
func BuildDir(packagePath string, options *build.Options) error {
	options.Normalize()
	s := build.NewSession(options)
	return s.BuildDir(filepath.Join(options.GOPATH, "src", packagePath), "main", "")
}

type packageBuilder struct {
	files      map[string][]byte
	options    *build.Options
	packageDir string
}

// BuildFile builds a package to Option.Target based on a single file
func BuildFile(r io.Reader, bo *build.Options) error {
	pb := NewBuilder(bo)
	err := pb.AddFile("main.go", r)
	if err != nil {
		return err
	}
	return pb.Build()
}

// New creates a new package builder. use it to add several files to
// a package before building it
func NewBuilder(bo *build.Options) *packageBuilder {
	bo.Normalize()

	return &packageBuilder{
		files:   map[string][]byte{},
		options: bo,
	}
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

	return BuildDir(b.packageDir, b.options)
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
