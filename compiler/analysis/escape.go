package analysis

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/types"
)

func EscapingObjects(n ast.Node, info *types.Info) map[*types.Var]bool {
	v := escapeAnalysis{
		info:       info,
		candidates: make(map[types.Object]bool),
		escaping:   make(map[*types.Var]bool),
	}
	ast.Walk(&v, n)
	return v.escaping
}

type escapeAnalysis struct {
	info       *types.Info
	candidates map[types.Object]bool
	escaping   map[*types.Var]bool
}

func (v *escapeAnalysis) Visit(node ast.Node) (w ast.Visitor) {
	// huge overapproximation
	switch n := node.(type) {
	case *ast.GenDecl:
		if n.Tok != token.VAR {
			return nil
		}
	case *ast.ValueSpec:
		for _, name := range n.Names {
			v.candidates[v.info.Defs[name].(*types.Var)] = true
		}
	case *ast.AssignStmt:
		if n.Tok == token.DEFINE {
			for _, name := range n.Lhs {
				if def := v.info.Defs[name.(*ast.Ident)]; def != nil {
					v.candidates[def.(*types.Var)] = true
				}
			}
		}
	case *ast.UnaryExpr:
		if n.Op == token.AND {
			switch v.info.Types[n.X].Type.Underlying().(type) {
			case *types.Struct, *types.Array:
				// always by reference
				return v
			default:
				return &escapingObjectCollector{v}
			}
		}
	case *ast.FuncLit:
		return &escapingObjectCollector{v}
	case *ast.ForStmt, *ast.RangeStmt:
		return nil
	}
	return v
}

type escapingObjectCollector struct {
	analysis *escapeAnalysis
}

func (v *escapingObjectCollector) Visit(node ast.Node) (w ast.Visitor) {
	if id, isIdent := node.(*ast.Ident); isIdent {
		if obj, ok := v.analysis.info.Uses[id].(*types.Var); ok {
			if v.analysis.candidates[obj] {
				v.analysis.escaping[obj] = true
			}
		}
	}
	return v
}
