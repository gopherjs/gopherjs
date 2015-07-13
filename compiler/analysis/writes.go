package analysis

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/types"
)

func Writes(n ast.Node, info *types.Info) map[*types.Var]struct{} {
	v := writesVisitor{
		info:   info,
		writes: make(map[*types.Var]struct{}),
	}
	ast.Walk(&v, n)
	return v.writes
}

type writesVisitor struct {
	info   *types.Info
	writes map[*types.Var]struct{}
}

func (v *writesVisitor) Visit(node ast.Node) (w ast.Visitor) {
	switch n := node.(type) {
	case *ast.AssignStmt:
		if n.Tok == token.ASSIGN {
			for _, lhs := range n.Lhs {
				if lhsVar, ok := varOf(lhs, v.info); ok {
					v.writes[lhsVar] = struct{}{}
				}
			}
		}
	case *ast.UnaryExpr:
		if n.Op == token.AND {
			if xVar, ok := varOf(n.X, v.info); ok {
				v.writes[xVar] = struct{}{}
			}
		}
	}
	return v
}

func varOf(expr ast.Expr, info *types.Info) (*types.Var, bool) {
	switch e := expr.(type) {
	case *ast.Ident:
		if e.Name != "_" {
			return info.Uses[e].(*types.Var), true
		}
	case *ast.SelectorExpr:
		sel, ok := info.Selections[e]
		if ok && !sel.Indirect() {
			return varOf(e.X, info)
		}
	case *ast.IndexExpr:
		if _, ok := info.Types[e.X].Type.Underlying().(*types.Array); ok {
			return varOf(e.X, info)
		}
	case *ast.SliceExpr:
		if _, ok := info.Types[e.X].Type.Underlying().(*types.Array); ok {
			return varOf(e.X, info)
		}
	}
	return nil, false
}
