package astutil

import (
	"go/ast"
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
