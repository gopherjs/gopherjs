package filter

import (
	"go/ast"

	"github.com/gopherjs/gopherjs/compiler/analysis"

	"golang.org/x/tools/go/types"
)

func setType(info *analysis.Info, t types.Type, e ast.Expr) ast.Expr {
	info.Types[e] = types.TypeAndValue{Type: t}
	return e
}

func newIdent(name string, t types.Type, info *analysis.Info) *ast.Ident {
	ident := ast.NewIdent(name)
	info.Types[ident] = types.TypeAndValue{Type: t}
	obj := types.NewVar(0, info.Pkg, name, t)
	info.Uses[ident] = obj
	return ident
}

func removeParens(e ast.Expr) ast.Expr {
	for {
		p, isParen := e.(*ast.ParenExpr)
		if !isParen {
			return e
		}
		e = p.X
	}
}
