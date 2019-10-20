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
	"go/token"
	"go/types"
	"log"
	"os"
	"path/filepath"

	"github.com/davecgh/go-spew/spew"
	build_v1 "github.com/gopherjs/gopherjs/build"
	"github.com/gopherjs/gopherjs/compiler"
	"github.com/neelance/sourcemap"
	"golang.org/x/tools/go/packages"
)

func init() {
	spew.Config.DisablePointerAddresses = true
	spew.Config.DisableMethods = true
}

type Options = build_v1.Options

// Session manages build process.
type Session struct {
	opts Options

	pkgs     map[string]*packages.Package
	archives map[string]*compiler.Archive
	types    map[string]*types.Package
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

	return &Session{
		opts:     opts,
		pkgs:     map[string]*packages.Package{},
		archives: map[string]*compiler.Archive{},
		types:    map[string]*types.Package{},
	}, nil
}

func (s *Session) Build(patterns ...string) ([]*compiler.Archive, error) {
	fmt.Printf("Building %s...\n", patterns)

	pkgs, err := s.load(patterns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load packages %q: %s", patterns, err)
	}

	archives, err := s.compile(pkgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to compile packages %s: %s", pkgs, err)
	}
	return archives, nil
}

func (s *Session) WriteCommandPackage(archive *compiler.Archive, pkgObj string) error {
	if err := os.MkdirAll(filepath.Dir(pkgObj), 0777); err != nil {
		return err
	}
	codeFile, err := os.Create(pkgObj)
	if err != nil {
		return err
	}
	defer codeFile.Close()

	sourceMapFilter := &compiler.SourceMapFilter{Writer: codeFile}
	if s.opts.CreateMapFile {
		m := &sourcemap.Map{File: filepath.Base(pkgObj)}
		mapFile, err := os.Create(pkgObj + ".map")
		if err != nil {
			return err
		}

		defer func() {
			m.WriteTo(mapFile)
			mapFile.Close()
			fmt.Fprintf(codeFile, "//# sourceMappingURL=%s.map\n", filepath.Base(pkgObj))
		}()

		sourceMapFilter.MappingCallback = build_v1.NewMappingCallback(m, s.opts.GOROOT, s.opts.GOPATH, s.opts.MapToLocalDisk)
	}

	deps, err := compiler.ImportDependencies(archive, s.loadAndCompile)
	if err != nil {
		return err
	}
	// deps := []*compiler.Archive{archive}
	return compiler.WriteProgramCode(deps, sourceMapFilter)
}

func (s *Session) load(patterns ...string) ([]*packages.Package, error) {
	log.Printf("Loading %s...", patterns)

	cfg := packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedSyntax | packages.NeedDeps | packages.NeedTypes,
		Fset: token.NewFileSet(),
		// TODO: This is different from the currently documented GopherJS behavior, which uses
		// GOOS=linux (or darvin) and GOARCH=js. We can't do exactly this because `go list` considers
		// this combination invalid. Since compiler uses 32-bit sizes, setting GOARCH=386 helps with
		// avoiding literal size mismatches down the road. However, this won't be necessary if we
		// implement natives VFS support and pruning for runtime package.
		//
		// With all that said, maybe its a good idea to use GOOS=js GOARCH=wasm? It seems to me implying
		// 64-bit sizes, though.
		Env: append(os.Environ(), "GOARCH=386"),
		// TODO: make sure to pass "js" build tag if we end up not using GOOS=js.
	}

	pkgs, err := packages.Load(&cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load %v: %s", patterns, err)
	}

	if count := packages.PrintErrors(pkgs); count > 0 {
		return nil, fmt.Errorf("encountered %d errors while loading %q", count, patterns)
	}

	// TODO: parseAndAugment()

	packages.Visit(pkgs, nil, func(p *packages.Package) {
		log.Printf("Putting %s package into cache...", p.ID)
		// TODO: Verify that this doesn't cause collisions with odd packages such as main or test.
		// If it does, consider using p.ID as a key.
		s.pkgs[p.PkgPath] = p
	})

	return pkgs, nil
}

func (s *Session) compile(pkgs ...*packages.Package) ([]*compiler.Archive, error) {
	// TODO: Check if recompilation is not required. Can we use go compiler's cache?
	log.Printf("Compiling %s...", pkgs)

	importCtx := &compiler.ImportContext{
		Packages: s.types,
		Import:   s.loadAndCompile,
	}

	archives := []*compiler.Archive{}

	for _, pkg := range pkgs {
		archive, err := compiler.Compile(pkg.PkgPath, pkg.Syntax, pkg.Fset, importCtx, s.opts.Minify)
		if err != nil {
			return nil, fmt.Errorf("failed to build package %q: %s", pkg, err)
		}
		archives = append(archives, archive)
		s.archives[archive.ImportPath] = archive
		log.Printf("Putting %q archive into cache...", archive.ImportPath)
	}

	return archives, nil
}

func (s *Session) loadAndCompile(path string) (*compiler.Archive, error) {
	if a, ok := s.archives[path]; ok {
		// We've already compiled this package during this session.
		return a, nil
	}

	var err error
	var pkgs []*packages.Package

	if p, ok := s.pkgs[path]; ok {
		// We've already loaded this package during this session, but haven't compiled it yet.
		pkgs = []*packages.Package{p}
	} else {
		// This is the first time we need this package during the session, let's load it.
		pkgs, err = s.load(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load %q: %s", path, err)
		}
		if count := len(pkgs); count != 1 {
			return nil, fmt.Errorf("s.load(%q) returned %d packages, expected 1", path, count)
		}
	}

	archives, err := s.compile(pkgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to compile %q: %s", path, err)
	}
	if count := len(archives); count != 1 {
		return nil, fmt.Errorf("s.compile(%q) returned %d archives, expected 1", path, count)
	}
	return archives[0], nil
}
