package analysis

import (
	"go/ast"

	"golang.org/x/tools/go/types"
)

func removeParens(e ast.Expr) ast.Expr {
	for {
		p, isParen := e.(*ast.ParenExpr)
		if !isParen {
			return e
		}
		e = p.X
	}
}

func isJsPackage(pkg *types.Package) bool {
	return pkg != nil && pkg.Path() == "github.com/gopherjs/gopherjs/js"
}

func isJsObject(t types.Type) bool {
	named, isNamed := t.(*types.Named)
	return isNamed && isJsPackage(named.Obj().Pkg()) && named.Obj().Name() == "Object"
}
