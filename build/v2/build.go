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
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	build_v1 "github.com/gopherjs/gopherjs/build"
	"github.com/gopherjs/gopherjs/compiler"
	"github.com/gopherjs/gopherjs/tracer"
	"github.com/neelance/sourcemap"
	"golang.org/x/tools/go/packages"
)

func init() {
	// spew.Config.DisablePointerAddresses = true
	// spew.Config.DisableMethods = true
}

type Options = build_v1.Options

// Session manages build process.
type Session struct {
	opts Options

	pkgs     map[string]*packages.Package
	archives map[string]*compiler.Archive
	types    map[string]*types.Package
	fset     *token.FileSet
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
		fset:     token.NewFileSet(),
	}, nil
}

func (s *Session) Build(patterns ...string) (_ []*compiler.Archive, err error) {
	defer tracer.Scope("Build(%s)\n", patterns).Leave(&err)

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

func (s *Session) load(patterns ...string) (_ []*packages.Package, err error) {
	defer tracer.Scope("load(%s)", patterns).Leave(&err)

	cfg := packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedSyntax |
			packages.NeedExportsFile |
			packages.NeedDeps,
		Fset: s.fset,
		// ParseFile: s.parseAndAugment,
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
		// TODO: Remove this workaround once http://golang.org/cl/205160 is accepted.
		if len(p.CompiledGoFiles) != len(p.Syntax) {
			tracer.Tracef("go/packages bug: syntax is not loaded, not caching %q", p.ID)
			return
		}
		tracer.Tracef("Putting %s package into cache...", p.ID)
		// TODO: Verify that this doesn't cause collisions with odd packages such as main or test.
		// If it does, consider using p.ID as a key.
		s.pkgs[p.PkgPath] = p
	})

	return pkgs, nil
}

func (s *Session) compile(pkgs ...*packages.Package) (_ []*compiler.Archive, err error) {
	// TODO: Check if recompilation is not required. Can we use go compiler's cache?
	defer tracer.Scope("compile(%s)", pkgs).Leave(&err)

	importCtx := &compiler.ImportContext{
		Packages: s.types,
		Import:   s.loadAndCompile,
	}

	archives := []*compiler.Archive{}

	for _, pkg := range pkgs {
		archive, err := compiler.Compile(pkg.PkgPath, pkg.Syntax, s.fset, importCtx, s.opts.Minify)
		if err != nil {
			return nil, fmt.Errorf("failed to build package %q: %s", pkg, err)
		}
		archives = append(archives, archive)
		s.archives[archive.ImportPath] = archive
		tracer.Tracef("Putting %q archive into cache...", archive.ImportPath)
	}

	return archives, nil
}

func (s *Session) loadAndCompile(path string) (_ *compiler.Archive, err error) {
	defer tracer.Scope("loadAndCompile(%q)", path).Leave(&err)
	if a, ok := s.archives[path]; ok {
		tracer.Tracef("Got %s from cache...", path)
		// We've already compiled this package during this session.
		return a, nil
	}

	var pkgs []*packages.Package

	if p, ok := s.pkgs[path]; ok {
		// We've already loaded this package during this session, but haven't compiled it yet.
		pkgs = []*packages.Package{p}
		tracer.Tracef("Package %s was pre-loaded earlier: %s", path, p)
	} else {
		tracer.Tracef("Loading package %s on demand from scratch...", path)
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

func (s *Session) parseAndAugment(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
	if s.pruned(filename) {
		return nil, nil
	}
	const mode = parser.AllErrors | parser.ParseComments
	file, err := parser.ParseFile(fset, filename, src, mode)
	if err != nil {
		return nil, err
	}
	return file, nil
}

var stdlibKeep = map[string]map[string]bool{
	"runtime": map[string]bool{"error.go": true},
}
var stdlibPrefix = path.Join(runtime.GOROOT(), "src")

func (s *Session) pruned(filename string) bool {
	dir, file := path.Split(filename)
	if !strings.HasPrefix(dir, stdlibPrefix) {
		return false
	}
	dir = strings.TrimPrefix(dir, stdlibPrefix)
	dir = strings.Trim(dir, "\\/")

	keep, ok := stdlibKeep[dir]
	if !ok {
		return false
	}

	if keep[file] {
		tracer.Tracef("Passed %s...", filename)
		return false
	}

	return true
}
