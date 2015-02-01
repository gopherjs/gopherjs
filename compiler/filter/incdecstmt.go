package filter

import (
	"go/ast"
	"go/token"

	"github.com/gopherjs/gopherjs/compiler/analysis"

	"golang.org/x/tools/go/exact"
	"golang.org/x/tools/go/types"
)

func IncDecStmt(stmt ast.Stmt, info *analysis.Info) ast.Stmt {
	if s, ok := stmt.(*ast.IncDecStmt); ok {
		t := info.Types[s.X].Type
		if iExpr, isIExpr := s.X.(*ast.IndexExpr); isIExpr {
			switch u := info.Types[iExpr.X].Type.Underlying().(type) {
			case *types.Array:
				t = u.Elem()
			case *types.Slice:
				t = u.Elem()
			case *types.Map:
				t = u.Elem()
			}
		}

		tok := token.ADD_ASSIGN
		if s.Tok == token.DEC {
			tok = token.SUB_ASSIGN
		}

		one := &ast.BasicLit{Kind: token.INT}
		info.Types[one] = types.TypeAndValue{Type: t, Value: exact.MakeInt64(1)}

		return &ast.AssignStmt{
			Lhs: []ast.Expr{s.X},
			Tok: tok,
			Rhs: []ast.Expr{one},
		}
	}
	return stmt
}
