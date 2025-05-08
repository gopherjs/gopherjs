package typeparams

import (
	"errors"
	"fmt"
	"go/token"
	"go/types"
)

// SignatureTypeParams returns receiver type params for methods, or function
// type params for standalone functions, or nil for non-generic functions and
// methods.
func SignatureTypeParams(sig *types.Signature) *types.TypeParamList {
	if tp := sig.RecvTypeParams(); tp != nil {
		return tp
	} else if tp := sig.TypeParams(); tp != nil {
		return tp
	} else {
		return nil
	}
}

// FindNestingFunc returns the function or method that the given object
// is nested in, or nil if the object was defined at the package level.
func FindNestingFunc(obj types.Object) *types.Func {
	objPos := obj.Pos()
	if objPos == token.NoPos {
		return nil
	}

	scope := obj.Parent()
	for scope != nil {
		// Iterate over all declarations in the scope.
		for _, name := range scope.Names() {
			decl := scope.Lookup(name)
			if fn, ok := decl.(*types.Func); ok {
				// Check if the object's position is within the function's scope.
				if objPos >= fn.Pos() && objPos <= fn.Scope().End() {
					return fn
				}
			}
		}
		scope = scope.Parent()
	}
	return nil
}

var (
	errInstantiatesGenerics = errors.New("instantiates generic type or function")
	errDefinesGenerics      = errors.New("defines generic type or function")
)

// HasTypeParams returns true if object defines type parameters.
//
// Note: this function doe not check if the object definition actually uses the
// type parameters, neither its own, nor from the outer scope.
func HasTypeParams(typ types.Type) bool {
	switch typ := typ.(type) {
	case *types.Signature:
		return typ.RecvTypeParams().Len() > 0 || typ.TypeParams().Len() > 0
	case *types.Named:
		return typ.TypeParams().Len() > 0
	default:
		return false
	}
}

// RequiresGenericsSupport returns an error if the type-checked code depends on
// generics support.
func RequiresGenericsSupport(info *types.Info) error {
	for ident := range info.Instances {
		// Any instantiation means dependency on generics.
		return fmt.Errorf("%w: %v", errInstantiatesGenerics, info.ObjectOf(ident))
	}

	for _, obj := range info.Defs {
		if obj == nil {
			continue
		}
		if HasTypeParams(obj.Type()) {
			return fmt.Errorf("%w: %v", errDefinesGenerics, obj)
		}
	}

	return nil
}
