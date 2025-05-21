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

// isGeneric will search all the given types and their subtypes for a
// *types.TypeParam. This will not check if a type could be generic,
// but if each instantiation is not completely concrete yet.
//
// This is useful to check for generics types like `X[B[T]]`, where
// `X` appears concrete because it is instantiated with the type argument `B[T]`,
// however the `T` inside `B[T]` is a type parameter making `X[B[T]]` a generic
// type since it required instantiation to a concrete type, e.g. `X[B[int]]`.
func isGeneric(typ ...types.Type) bool {
	var containsTypeParam func(t types.Type) bool

	foreach := func(count int, getter func(index int) types.Type) bool {
		for i := 0; i < count; i++ {
			if containsTypeParam(getter(i)) {
				return true
			}
		}
		return false
	}

	trace := `` // TODO(grantnelson-wf): REMOVE
	seen := make(map[types.Type]struct{})
	containsTypeParam = func(t types.Type) (result bool) {
		defer func() {
			if result {
				trace += fmt.Sprintf("\tin %v\n", t) // TODO(grantnelson-wf): REMOVE
			}
		}()

		if _, ok := seen[t]; ok {
			return false
		}
		seen[t] = struct{}{}

		switch t := t.(type) {
		case *types.TypeParam:
			return true
		case *types.Named:
			return t.TypeParams().Len() != t.TypeArgs().Len() ||
				foreach(t.TypeArgs().Len(), func(i int) types.Type { return t.TypeArgs().At(i) }) ||
				containsTypeParam(t.Underlying())
		case *types.Struct:
			return foreach(t.NumFields(), func(i int) types.Type { return t.Field(i).Type() })
		case *types.Interface:
			return foreach(t.NumMethods(), func(i int) types.Type { return t.Method(i).Type() })
		case *types.Signature:
			return foreach(t.Params().Len(), func(i int) types.Type { return t.Params().At(i).Type() }) ||
				foreach(t.Results().Len(), func(i int) types.Type { return t.Results().At(i).Type() })
		case *types.Map:
			return containsTypeParam(t.Key()) || containsTypeParam(t.Elem())
		case interface{ Elem() types.Type }:
			// Handles *types.Pointer, *types.Slice, *types.Array, *types.Chan
			return containsTypeParam(t.Elem())
		default:
			// Other types (e.g., basic types) do not contain type parameters.
			return false
		}
	}

	result := foreach(len(typ), func(i int) types.Type { return typ[i] })
	if result {
		fmt.Println("isGeneric:\n" + trace) // TODO(grantnelson-wf): REMOVE
	}
	return result
}
