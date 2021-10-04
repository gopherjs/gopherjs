package astutil

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

func RemoveParens(e ast.Expr) ast.Expr {
	for {
		p, isParen := e.(*ast.ParenExpr)
		if !isParen {
			return e
		}
		e = p.X
	}
}

func SetType(info *types.Info, t types.Type, e ast.Expr) ast.Expr {
	info.Types[e] = types.TypeAndValue{Type: t}
	return e
}

func NewIdent(name string, t types.Type, info *types.Info, pkg *types.Package) *ast.Ident {
	ident := ast.NewIdent(name)
	info.Types[ident] = types.TypeAndValue{Type: t}
	obj := types.NewVar(0, pkg, name, t)
	info.Uses[ident] = obj
	return ident
}

func IsTypeExpr(expr ast.Expr, info *types.Info) bool {
	switch e := expr.(type) {
	case *ast.ArrayType, *ast.ChanType, *ast.FuncType, *ast.InterfaceType, *ast.MapType, *ast.StructType:
		return true
	case *ast.StarExpr:
		return IsTypeExpr(e.X, info)
	case *ast.Ident:
		_, ok := info.Uses[e].(*types.TypeName)
		return ok
	case *ast.SelectorExpr:
		_, ok := info.Uses[e.Sel].(*types.TypeName)
		return ok
	case *ast.ParenExpr:
		return IsTypeExpr(e.X, info)
	default:
		return false
	}
}

func ImportsUnsafe(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp.Path.Value == `"unsafe"` {
			return true
		}
	}
	return false
}

// FuncKey returns a string, which uniquely identifies a top-level function or
// method in a package.
func FuncKey(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return d.Name.Name
	}
	recv := d.Recv.List[0].Type
	if star, ok := recv.(*ast.StarExpr); ok {
		recv = star.X
	}
	return recv.(*ast.Ident).Name + "." + d.Name.Name
}

// PruneOriginal returns true if gopherjs:prune-original directive is present
// before a function decl.
//
// `//gopherjs:prune-original` is a GopherJS-specific directive, which can be
// applied to functions in native overlays and will instruct the augmentation
// logic to delete the body of a standard library function that was replaced.
// This directive can be used to remove code that would be invalid in GopherJS,
// such as code expecting ints to be 64-bit. It should be used with caution
// since it may create unused imports in the original source file.
func PruneOriginal(d *ast.FuncDecl) bool {
	if d.Doc == nil {
		return false
	}
	for _, c := range d.Doc.List {
		if strings.HasPrefix(c.Text, "//gopherjs:prune-original") {
			return true
		}
	}
	return false
}

// FindLoopStmt tries to find the loop statement among the AST nodes in the
// |stack| that corresponds to the break/continue statement represented by
// branch.
//
// This function is label-aware and assumes the code was successfully
// type-checked.
func FindLoopStmt(stack []ast.Node, branch *ast.BranchStmt, typeInfo *types.Info) ast.Stmt {
	if branch.Tok != token.CONTINUE && branch.Tok != token.BREAK {
		panic(fmt.Errorf("FindLoopStmt() must be used with a break or continue statement only, got: %v", branch))
	}

	for i := len(stack) - 1; i >= 0; i-- {
		n := stack[i]

		if branch.Label != nil {
			// For a labelled continue the loop will always be in a labelled statement.
			referencedLabel := typeInfo.Uses[branch.Label].(*types.Label)
			labelStmt, ok := n.(*ast.LabeledStmt)
			if !ok {
				continue
			}
			if definedLabel := typeInfo.Defs[labelStmt.Label]; definedLabel != referencedLabel {
				continue
			}
			n = labelStmt.Stmt
		}

		switch s := n.(type) {
		case *ast.RangeStmt, *ast.ForStmt:
			return s.(ast.Stmt)
		}
	}

	// This should never happen in a source that passed type checking.
	panic(fmt.Errorf("continue/break statement %v doesn't have a matching loop statement among ancestors", branch))
}

// EndsWithReturn returns true if the last effective statement is a "return".
func EndsWithReturn(stmts []ast.Stmt) bool {
	if len(stmts) == 0 {
		return false
	}
	last := stmts[len(stmts)-1]
	switch l := last.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.LabeledStmt:
		return EndsWithReturn([]ast.Stmt{l.Stmt})
	case *ast.BlockStmt:
		return EndsWithReturn(l.List)
	default:
		return false
	}
}
