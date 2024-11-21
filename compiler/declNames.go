package compiler

import (
	"go/types"

	"github.com/gopherjs/gopherjs/compiler/internal/symbol"
	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
)

// importDeclFullName returns a unique name for an import declaration.
// This import name may be duplicated in different packages if they both
// import the same package, they are only unique per package.
func importDeclFullName(importedPkg *types.Package) string {
	return `import:` + importedPkg.Path()
}

// varDeclFullName returns a name for a package-level variable declaration.
// This var name only references the first named variable in an assignment.
// If no variables are named, the name is `var:blank` and not unique.
func varDeclFullName(init *types.Initializer) string {
	for _, lhs := range init.Lhs {
		if lhs.Name() != `_` {
			return `var:` + symbol.New(lhs).String()
		}
	}
	return `var:blank`
}

// funcVarDeclFullName returns a name for a package-level variable
// that is used for a function (without a receiver) declaration.
// The name is unique unless the function is an `init` function.
// If the function is generic, this declaration name is also for the list
// of instantiations of the function.
func funcVarDeclFullName(o *types.Func) string {
	return `funcVar:` + symbol.New(o).String()
}

// mainFuncFullName returns the name for the declaration used to invoke the
// main function of the program. There should only be one decl with this name.
func mainFuncDeclFullName() string {
	return `init:main`
}

// funcDeclFullName returns a name for a package-level function
// declaration for the given instance of a function.
// The name is unique except unless the function is an `init` function.
func funcDeclFullName(inst typeparams.Instance) string {
	return `func:` + inst.String()
}

// typeVarDeclFullName returns a unique name for a package-level variable
// that is used for a named type declaration.
// If the type is generic, this declaration name is also for the list
// of instantiations of the type.
func typeVarDeclFullName(o *types.TypeName) string {
	return `typeVar:` + symbol.New(o).String()
}

// typeDeclFullName returns a unique name for a package-level type declaration
// for the given instance of a type. Names are only unique per package.
func typeDeclFullName(inst typeparams.Instance) string {
	return `type:` + inst.String()
}

// anonTypeDeclFullName returns a unique name for a package-level type
// declaration for an anonymous type. Names are only unique per package.
// These names are generated for types that are not named in the source code.
func anonTypeDeclFullName(o types.Object) string {
	return `anonType:` + symbol.New(o).String()
}
