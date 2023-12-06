// Package srctesting contains common helpers for unit testing source code
// analysis and transformation.
package srctesting

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

// Parse source from the string and return complete AST.
//
// Assumes source file name `test.go`. Fails the test on parsing error.
func Parse(t *testing.T, fset *token.FileSet, src string) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(fset, "test.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse test source: %s", err)
	}
	return f
}

// Check type correctness of the provided AST.
//
// Assumes "test" package import path. Fails the test if type checking fails.
// Provided AST is expected not to have any imports.
func Check(t *testing.T, fset *token.FileSet, files ...*ast.File) (*types.Info, *types.Package) {
	t.Helper()
	typesInfo := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
	}
	config := &types.Config{
		Sizes: &types.StdSizes{WordSize: 4, MaxAlign: 8},
	}
	typesPkg, err := config.Check("test", fset, files, typesInfo)
	if err != nil {
		t.Fatalf("Filed to type check test source: %s", err)
	}
	return typesInfo, typesPkg
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
	fset := token.NewFileSet()
	file := Parse(t, fset, src)
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
