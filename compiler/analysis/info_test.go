package analysis

import (
	"go/ast"
	"go/token"
	"go/types"
	"testing"

	"github.com/gopherjs/gopherjs/internal/srctesting"
)

// See: https://github.com/gopherjs/gopherjs/issues/955.
func TestBlockingFunctionLiteral(t *testing.T) {
	src := `
package test

func blocking() {
	c := make(chan bool)
	<-c
}

func indirectlyBlocking() {
	func() { blocking() }()
}

func directlyBlocking() {
	func() {
		c := make(chan bool)
		<-c
	}()
}

func notBlocking() {
	func() { println() } ()
}
`
	fset := token.NewFileSet()
	file := srctesting.Parse(t, fset, src)
	typesInfo, typesPkg := srctesting.Check(t, fset, file)

	pkgInfo := AnalyzePkg([]*ast.File{file}, fset, typesInfo, typesPkg, func(f *types.Func) bool {
		panic("isBlocking() should be never called for imported functions in this test.")
	})

	assertBlocking(t, file, pkgInfo, "blocking")
	assertBlocking(t, file, pkgInfo, "indirectlyBlocking")
	assertBlocking(t, file, pkgInfo, "directlyBlocking")
	assertNotBlocking(t, file, pkgInfo, "notBlocking")
}

func assertBlocking(t *testing.T, file *ast.File, pkgInfo *Info, funcName string) {
	typesFunc := getTypesFunc(t, file, pkgInfo, funcName)
	if !pkgInfo.IsBlocking(typesFunc) {
		t.Errorf("Got: %q is not blocking. Want: %q is blocking.", typesFunc, typesFunc)
	}
}

func assertNotBlocking(t *testing.T, file *ast.File, pkgInfo *Info, funcName string) {
	typesFunc := getTypesFunc(t, file, pkgInfo, funcName)
	if pkgInfo.IsBlocking(typesFunc) {
		t.Errorf("Got: %q is blocking. Want: %q is not blocking.", typesFunc, typesFunc)
	}
}

func getTypesFunc(t *testing.T, file *ast.File, pkgInfo *Info, funcName string) *types.Func {
	obj := file.Scope.Lookup(funcName)
	if obj == nil {
		t.Fatalf("Declaration of %q is not found in the AST.", funcName)
	}
	decl, ok := obj.Decl.(*ast.FuncDecl)
	if !ok {
		t.Fatalf("Got: %q is %v. Want: a function declaration.", funcName, obj.Kind)
	}
	blockingType, ok := pkgInfo.Defs[decl.Name]
	if !ok {
		t.Fatalf("No type information is found for %v.", decl.Name)
	}
	return blockingType.(*types.Func)
}
