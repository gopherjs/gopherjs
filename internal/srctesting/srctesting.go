// Package srctesting contains common helpers for unit testing source code
// analysis and transformation.
package srctesting

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

// Fixture provides utilities for parsing and type checking Go code in tests.
type Fixture struct {
	T        *testing.T
	FileSet  *token.FileSet
	Info     *types.Info
	Packages map[string]*types.Package
}

func newInfo() *types.Info {
	return &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
		Instances:  make(map[*ast.Ident]types.Instance),
	}
}

// New creates a fresh Fixture.
func New(t *testing.T) *Fixture {
	return &Fixture{
		T:        t,
		FileSet:  token.NewFileSet(),
		Info:     newInfo(),
		Packages: map[string]*types.Package{},
	}
}

// Parse source from the string and return complete AST.
func (f *Fixture) Parse(name, src string) *ast.File {
	f.T.Helper()
	file, err := parser.ParseFile(f.FileSet, name, src, parser.ParseComments)
	if err != nil {
		f.T.Fatalf("Failed to parse test source: %s", err)
	}
	return file
}

// Check type correctness of the provided AST.
//
// Fails the test if type checking fails. Provided AST is expected not to have
// any imports. If f.Info is nil, it will create a new types.Info instance
// to store type checking results and return it, otherwise f.Info is used.
func (f *Fixture) Check(importPath string, files ...*ast.File) (*types.Info, *types.Package) {
	f.T.Helper()
	config := &types.Config{
		Sizes:    &types.StdSizes{WordSize: 4, MaxAlign: 8},
		Importer: f,
	}
	info := f.Info
	if info == nil {
		info = newInfo()
	}
	pkg, err := config.Check(importPath, f.FileSet, files, info)
	if err != nil {
		f.T.Fatalf("Failed to type check test source: %s", err)
	}
	f.Packages[importPath] = pkg
	return info, pkg
}

// Import implements types.Importer.
func (f *Fixture) Import(path string) (*types.Package, error) {
	pkg, ok := f.Packages[path]
	if !ok {
		return nil, fmt.Errorf("missing type info for package %q", path)
	}
	return pkg, nil
}

// ParseFuncDecl parses source with a single function defined and returns the
// function AST.
//
// Fails the test if there isn't exactly one function declared in the source.
func ParseFuncDecl(t *testing.T, src string) *ast.FuncDecl {
	t.Helper()
	decl := ParseDecl(t, src)
	fdecl, ok := decl.(*ast.FuncDecl)
	if !ok {
		t.Fatalf("Got %T decl, expected *ast.FuncDecl", decl)
	}
	return fdecl
}

// ParseDecl parses source with a single declaration and
// returns that declaration AST.
//
// Fails the test if there isn't exactly one declaration in the source.
func ParseDecl(t *testing.T, src string) ast.Decl {
	t.Helper()
	file := New(t).Parse("test.go", src)
	if l := len(file.Decls); l != 1 {
		t.Fatalf(`Got %d decls in the sources, expected exactly 1`, l)
	}
	return file.Decls[0]
}

// ParseSpec parses source with a single declaration containing
// a single specification and returns that specification AST.
//
// Fails the test if there isn't exactly one declaration and
// one specification in the source.
func ParseSpec(t *testing.T, src string) ast.Spec {
	t.Helper()
	decl := ParseDecl(t, src)
	gdecl, ok := decl.(*ast.GenDecl)
	if !ok {
		t.Fatalf("Got %T decl, expected *ast.GenDecl", decl)
	}
	if l := len(gdecl.Specs); l != 1 {
		t.Fatalf(`Got %d spec in the sources, expected exactly 1`, l)
	}
	return gdecl.Specs[0]
}

// Format AST node into a string.
//
// The node type must be *ast.File, *printer.CommentedNode, []ast.Decl,
// []ast.Stmt, or assignment-compatible to ast.Expr, ast.Decl, ast.Spec, or
// ast.Stmt.
func Format(t *testing.T, fset *token.FileSet, node any) string {
	t.Helper()
	buf := &bytes.Buffer{}
	if err := format.Node(buf, fset, node); err != nil {
		t.Fatalf("Failed to format AST node %T: %s", node, err)
	}
	return buf.String()
}

// LookupObj returns a top-level object with the given name.
//
// Methods can be referred to as RecvTypeName.MethodName.
func LookupObj(pkg *types.Package, name string) types.Object {
	path := strings.Split(name, ".")
	scope := pkg.Scope()
	var obj types.Object

	for len(path) > 0 {
		obj = scope.Lookup(path[0])
		path = path[1:]

		if fun, ok := obj.(*types.Func); ok {
			scope = fun.Scope()
			continue
		}

		// If we are here, the latest object is a named type. If there are more path
		// elements left, they must refer to field or method.
		if len(path) > 0 {
			obj, _, _ = types.LookupFieldOrMethod(obj.Type(), true, obj.Pkg(), path[0])
			path = path[1:]
			if fun, ok := obj.(*types.Func); ok {
				scope = fun.Scope()
			}
		}
	}
	return obj
}

type Source struct {
	Name     string
	Contents []byte
}

// ParseSources parses the given source files and returns the root package
// that contains the given source files.
//
// The source file should all be from the same package as the files for the
// root package. At least one source file must be given.
// The root package's path will be `command-line-arguments`.
//
// The auxiliary files can be for different packages but should have paths
// added to the source name so that they can be grouped together by package.
// To import an auxiliary package, the path should be prepended by
// `github.com/gopherjs/gopherjs/compiler`.
func ParseSources(t *testing.T, sourceFiles []Source, auxFiles []Source) *packages.Package {
	t.Helper()
	const mode = packages.NeedName |
		packages.NeedFiles |
		packages.NeedImports |
		packages.NeedDeps |
		packages.NeedTypes |
		packages.NeedSyntax

	dir, err := filepath.Abs(`./`)
	if err != nil {
		t.Fatal(`error getting working directory:`, err)
	}

	patterns := make([]string, len(sourceFiles))
	overlay := make(map[string][]byte, len(sourceFiles))
	for i, src := range sourceFiles {
		filename := src.Name
		patterns[i] = filename
		absName := filepath.Join(dir, filename)
		overlay[absName] = []byte(src.Contents)
	}
	for _, src := range auxFiles {
		absName := filepath.Join(dir, src.Name)
		overlay[absName] = []byte(src.Contents)
	}

	config := &packages.Config{
		Mode:    mode,
		Overlay: overlay,
		Dir:     dir,
	}

	pkgs, err := packages.Load(config, patterns...)
	if err != nil {
		t.Fatal(`error loading packages:`, err)
	}

	hasErrors := false
	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		for _, err := range pkg.Errors {
			hasErrors = true
			t.Error(err)
		}
	})
	if hasErrors {
		t.FailNow()
	}

	if len(pkgs) != 1 {
		t.Fatal(`expected one and only one root package but got`, len(pkgs))
	}
	return pkgs[0]
}

// GetNodeAtLineNo returns the first node of type N that starts on the given
// line in the given file. This helps lookup nodes that aren't named but
// are needed by a specific test.
func GetNodeAtLineNo[N ast.Node](file *ast.File, fSet *token.FileSet, lineNo int) N {
	var node N
	keepLooking := true
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil || !keepLooking {
			return false
		}
		nodeLine := fSet.Position(n.Pos()).Line
		switch {
		case nodeLine < lineNo:
			// We haven't reached the line yet, so check if we can skip over
			// this whole node or if we should look inside it.
			return fSet.Position(n.End()).Line >= lineNo
		case nodeLine > lineNo:
			// We went past it without finding it, so stop looking.
			keepLooking = false
			return false
		default: // nodeLine == lineNo
			if n, ok := n.(N); ok {
				node = n
				keepLooking = false
			}
			return keepLooking
		}
	})
	return node
}
