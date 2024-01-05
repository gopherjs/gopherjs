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
	"strings"
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
		Instances:  make(map[*ast.Ident]types.Instance),
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
	fset := token.NewFileSet()
	file := Parse(t, fset, src)
	if l := len(file.Decls); l != 1 {
		t.Fatalf("Got %d decls in the sources, expected exactly 1", l)
	}
	fdecl, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok {
		t.Fatalf("Got %T decl, expected *ast.FuncDecl", file.Decls[0])
	}
	return fdecl
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
