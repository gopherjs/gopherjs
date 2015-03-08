package analysis

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/types"
)

func EscapingObjects(n ast.Node, info *types.Info) map[*types.Var]bool {
	v := escapeAnalysis{
		info:     info,
		escaping: make(map[*types.Var]bool),
		topScope: info.Scopes[n],
	}
	ast.Walk(&v, n)
	return v.escaping
}

type escapeAnalysis struct {
	info     *types.Info
	escaping map[*types.Var]bool
	topScope *types.Scope
}

func (v *escapeAnalysis) Visit(node ast.Node) (w ast.Visitor) {
	// huge overapproximation
	switch n := node.(type) {
	case *ast.UnaryExpr:
		if n.Op == token.AND {
			if _, ok := n.X.(*ast.Ident); ok {
				return &escapingObjectCollector{v, nil}
			}
		}
	case *ast.FuncLit:
		return &escapingObjectCollector{v, v.info.Scopes[n.Type]}
	case *ast.ForStmt, *ast.RangeStmt:
		return nil
	}
	return v
}

type escapingObjectCollector struct {
	analysis    *escapeAnalysis
	bottomScope *types.Scope
}

func (v *escapingObjectCollector) Visit(node ast.Node) (w ast.Visitor) {
	if id, ok := node.(*ast.Ident); ok {
		if obj, ok := v.analysis.info.Uses[id].(*types.Var); ok {
			switch obj.Type().Underlying().(type) {
			case *types.Struct, *types.Array:
				// always by reference
				return nil
			}

			for s := obj.Parent(); s != nil; s = s.Parent() {
				if s == v.bottomScope {
					break
				}
				if s == v.analysis.topScope {
					v.analysis.escaping[obj] = true
					break
				}
			}
		}
	}
	return v
}
