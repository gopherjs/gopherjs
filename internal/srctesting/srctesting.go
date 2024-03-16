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
	"strings"
	"testing"
)

// Fixture provides utilities for parsing and type checking Go code in tests.
type Fixture struct {
	T        *testing.T
	FileSet  *token.FileSet
	Info     *types.Info
	Packages map[string]*types.Package
}

// New creates a fresh Fixture.
func New(t *testing.T) *Fixture {
	return &Fixture{
		T:       t,
		FileSet: token.NewFileSet(),
		Info: &types.Info{
			Types:      make(map[ast.Expr]types.TypeAndValue),
			Defs:       make(map[*ast.Ident]types.Object),
			Uses:       make(map[*ast.Ident]types.Object),
			Implicits:  make(map[ast.Node]types.Object),
			Selections: make(map[*ast.SelectorExpr]*types.Selection),
			Scopes:     make(map[ast.Node]*types.Scope),
			Instances:  make(map[*ast.Ident]types.Instance),
		},
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
// any imports.
func (f *Fixture) Check(importPath string, files ...*ast.File) (*types.Info, *types.Package) {
	f.T.Helper()
	config := &types.Config{
		Sizes:    &types.StdSizes{WordSize: 4, MaxAlign: 8},
		Importer: f,
	}
	pkg, err := config.Check(importPath, f.FileSet, files, f.Info)
	if err != nil {
		f.T.Fatalf("Filed to type check test source: %s", err)
	}
	f.Packages[importPath] = pkg
	return f.Info, pkg
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
		}
	}
	return obj
}
