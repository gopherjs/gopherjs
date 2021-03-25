package compiler

import "go/types"

// GoLinkname describes a go:linkname compiler directive found in the source code.
//
// GopherJS treats these directives in a way that resembles a symbolic link,
// where for a single given symbol implementation there may be zero or more
// symbols referencing it. This is subtly different from the upstream Go
// implementation, which simply overrides symbol name the linker will use.
//
// The following directive format is supported:
// //go:linkname <localname> <importpath>.<name>
//
// Currently only package-level functions are supported for use with this
// directive.
//
// The compiler infers whether the local symbols is meant to be a reference or
// implementation based on whether the local symbol has implementation or not.
// The program behavior when neither or both symbols have implementation is
// undefined.
//
// Both reference and implementation symbols are referred to by a fully qualified
// name (see go/types.ObjectString()).
type GoLinkname struct {
	Implementation SymName
	Reference      SymName
}

// SymName uniquely identifies a named submol within a program.
//
// This is a logical equivalent of a symbol name used by traditional linkers.
// The following properties should hold true:
//
//  - Each named symbol within a program has a unique SymName.
//  - Similarly named methods of different types will have different symbol names.
//  - The string representation is opaque and should not be attempted to reversed
//    to a struct form.
type SymName struct {
	PkgPath string // Full package import path.
	Name    string // Symbol name.
}

// NewSymName constructs SymName for a given named symbol.
func NewSymName(o types.Object) SymName {
	if fun, ok := o.(*types.Func); ok {
		sig := fun.Type().(*types.Signature)
		if recv := sig.Recv(); recv != nil {
			// Special case: disambiguate names for different types' methods.
			return SymName{
				PkgPath: o.Pkg().Path(),
				Name:    recv.Type().(*types.Named).Obj().Name() + "." + o.Name(),
			}
		}
	}
	return SymName{
		PkgPath: o.Pkg().Path(),
		Name:    o.Name(),
	}
}

func (n SymName) String() string { return n.PkgPath + "." + n.Name }
