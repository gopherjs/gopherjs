package compiler

import (
	"fmt"
	"go/token"
	"go/types"
	"path"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/internal/symbol"
	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
)

const (
	varDeclNotUniqueFullName      = `var:blank`
	funcVarDeclNotUniqueFullName  = `funcVar:init`
	mainFuncDeclNotUniqueFullName = `init:main`
	funcDeclNotUniqueFullName     = `func:init`
	typeDeclNestedPrefix          = `type:nested:`
)

// importDeclFullName returns a unique name for an import declaration.
// This import name may be duplicated in different packages if they both
// import the same package, they are only unique per package.
func importDeclFullName(importedPkg *types.Package) string {
	return `import:` + importedPkg.Path()
}

// varDeclFullName returns a name for a package-level variable declaration.
// This var name only references the first named variable in an assignment.
//
// If no variables are named, this will attempt to create a unique name
// using the integer value of the first valid position of the blank variables.
// Otherwise the name is not unique (i.e. varDeclNotUniqueFullName).
func varDeclFullName(init *types.Initializer, fSet *token.FileSet) string {
	for _, lhs := range init.Lhs {
		if lhs.Name() != `_` {
			return `var:` + symbol.New(lhs).String()
		}
	}

	for _, lhs := range init.Lhs {
		if pos := lhs.Pos(); pos.IsValid() {
			return `var:blank` + declFullNameDiscriminator(pos, fSet)
		}
	}

	return varDeclNotUniqueFullName
}

// funcVarDeclFullName returns a name for a package-level variable
// that is used for a function (without a receiver) declaration.
// If the function is generic, this declaration name is for the list
// of instantiations of the function.
//
// The name is unique unless the function is an `init` function.
// If the name is for an `init` function, the position of the function
// is used to create a unique name if it is valid,
// otherwise the name is not unique (i.e. funcVarDeclNotUniqueFullName).
func funcVarDeclFullName(o *types.Func, fSet *token.FileSet) string {
	name := `funcVar:` + symbol.New(o).String()
	if o.Name() == `init` {
		name += declFullNameDiscriminator(o.Pos(), fSet)
	}
	// In the case of an `init` function with an invalid position,
	// the name is not unique and will be equal to funcVarDeclNotUnqueName.
	return name
}

// mainFuncDeclFullName returns the name for the declaration used to invoke the
// main function of the program. There should only be one decl with this name.
// This will always return mainFuncDeclNotUniqueFullName.
func mainFuncDeclFullName() string {
	return mainFuncDeclNotUniqueFullName
}

// funcDeclFullName returns a name for a package-level function
// declaration for the given instance of a function.
// The name is unique unless the function is an `init` function
// then it tries to use the position of the function to create a unique name.
// Otherwise the name is not unique (i.e. funcDeclNotUniqueFullName).
func funcDeclFullName(inst typeparams.Instance, fSet *token.FileSet) string {
	name := `func:` + inst.String()
	if inst.IsTrivial() && inst.Object.Name() == `init` {
		name += declFullNameDiscriminator(inst.Object.Pos(), fSet)
	}
	// In the case of an `init` function with an invalid position,
	// the name is not unique and will be equal to funcDeclNotUniqueFullName.
	return name
}

// typeVarDeclFullName returns a unique name for a package-level variable
// that is used for a named type declaration or a named nested type declaration.
// If the type is generic, this declaration name is also for the list
// of instantiations of the type.
func typeVarDeclFullName(o *types.TypeName) string {
	return `typeVar:` + symbol.New(o).String()
}

// typeDeclFullName returns a unique name for a package-level type declaration
// for the given instance of a type. Names are only unique per package
// unless the type is a nested type then the name is only unique per the
// function or method it is declared in. In that case, the position of the
// declaration is used to try to create a unique name.
// If the position is invalid, the nested name may be unique or not,
// we cannot guarantee uniqueness in that case.
func typeDeclFullName(inst typeparams.Instance, fSet *token.FileSet) string {
	if typeparams.FindNestingFunc(inst.Object) != nil {
		return typeDeclNestedPrefix + inst.String() +
			declFullNameDiscriminator(inst.Object.Pos(), fSet)
	}
	return `type:` + inst.String()
}

// anonTypeDeclFullName returns a unique name for a package-level type
// declaration for an anonymous type. Names are only unique per package.
// These names are generated for types that are not named in the source code.
func anonTypeDeclFullName(o types.Object) string {
	return `anonType:` + symbol.New(o).String()
}

// declFullNameDiscriminator returns a string representing the position of a
// declaration that is used to differentiate declarations with the same name.
// Since the package path will already be part of the full name, only the file,
// line, and column are used.
// If position is invalid, an empty string is returned.
func declFullNameDiscriminator(pos token.Pos, fSet *token.FileSet) string {
	if !pos.IsValid() {
		return ``
	}
	p := fSet.Position(pos)
	return fmt.Sprintf("@%s:%d:%d", path.Base(p.Filename), p.Line, p.Column)
}

// isUnqueDeclFullName reports whether the given declaration full name is unique.
// A unique declaration name can be safely cached and reused.
func isUnqueDeclFullName(name string) bool {
	switch name {
	case ``, varDeclNotUniqueFullName, funcVarDeclNotUniqueFullName,
		mainFuncDeclNotUniqueFullName, funcDeclNotUniqueFullName:
		return false // not unique since it equals one of the known non-unique names
	}

	// Check if the name is for a nested type declaration without a position discriminator.
	return !strings.HasPrefix(name, typeDeclNestedPrefix) || strings.Contains(name, "@")
}
