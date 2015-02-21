package analysis

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/exact"
	"golang.org/x/tools/go/types"
)

func BoolValue(expr ast.Expr, info *types.Info) (bool, bool) {
	v := info.Types[expr].Value
	if v != nil && v.Kind() == exact.Bool {
		return exact.BoolVal(v), true
	}
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		switch e.Op {
		case token.LAND:
			if b, ok := BoolValue(e.X, info); ok {
				if !b {
					return false, true
				}
				return BoolValue(e.Y, info)
			}
		case token.LOR:
			if b, ok := BoolValue(e.X, info); ok {
				if b {
					return true, true
				}
				return BoolValue(e.Y, info)
			}
		}
	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			if b, ok := BoolValue(e.X, info); ok {
				return !b, true
			}
		}
	case *ast.ParenExpr:
		return BoolValue(e.X, info)
	}
	return false, false
}
