package symbol

import (
	"go/types"
	"strings"
)

// Name uniquely identifies a named symbol within a program.
//
// This is a logical equivalent of a symbol name used by traditional linkers.
// The following properties should hold true:
//
//   - Each named symbol within a program has a unique Name.
//   - Similarly named methods of different types will have different symbol names.
//   - The string representation is opaque and should not be attempted to reversed
//     to a struct form.
type Name struct {
	PkgPath string // Full package import path.
	Name    string // Symbol name.
}

// New constructs SymName for a given named symbol.
func New(o types.Object) Name {
	if fun, ok := o.(*types.Func); ok {
		sig := fun.Type().(*types.Signature)
		if recv := sig.Recv(); recv != nil {
			// Special case: disambiguate names for different types' methods.
			typ := recv.Type()
			if ptr, ok := typ.(*types.Pointer); ok {
				return Name{
					PkgPath: o.Pkg().Path(),
					Name:    "(*" + ptr.Elem().(*types.Named).Obj().Name() + ")." + o.Name(),
				}
			}
			return Name{
				PkgPath: o.Pkg().Path(),
				Name:    typ.(*types.Named).Obj().Name() + "." + o.Name(),
			}
		}
	}
	return Name{
		PkgPath: o.Pkg().Path(),
		Name:    o.Name(),
	}
}

func (n Name) String() string { return n.PkgPath + "." + n.Name }

func (n Name) IsMethod() (recv string, method string, ok bool) {
	pos := strings.IndexByte(n.Name, '.')
	if pos == -1 {
		return
	}
	recv, method, ok = n.Name[:pos], n.Name[pos+1:], true
	size := len(recv)
	if size > 2 && recv[0] == '(' && recv[size-1] == ')' {
		recv = recv[1 : size-1]
	}
	return
}
