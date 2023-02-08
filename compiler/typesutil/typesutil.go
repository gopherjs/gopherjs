package typesutil

import (
	"fmt"
	"go/types"

	"golang.org/x/tools/go/types/typeutil"
)

// IsJsPackage returns is the package is github.com/gopherjs/gopherjs/js.
func IsJsPackage(pkg *types.Package) bool {
	return pkg != nil && pkg.Path() == "github.com/gopherjs/gopherjs/js"
}

// IsJsObject returns true if the type represents a pointer to github.com/gopherjs/gopherjs/js.Object.
func IsJsObject(t types.Type) bool {
	ptr, isPtr := t.(*types.Pointer)
	if !isPtr {
		return false
	}
	named, isNamed := ptr.Elem().(*types.Named)
	return isNamed && IsJsPackage(named.Obj().Pkg()) && named.Obj().Name() == "Object"
}

// AnonymousTypes maintains a mapping between anonymous types encountered in a
// Go program to equivalent synthetic names types GopherJS generated from them.
//
// This enables a runtime performance optimization where different instances of
// the same anonymous type (e.g. in expression `x := map[int]string{}`) don't
// need to initialize type information (e.g. `$mapType($Int, $String)`) every
// time, but reuse the single synthesized type (e.g. `mapType$1`).
type AnonymousTypes struct {
	m     typeutil.Map
	order []*types.TypeName
}

// Get returns the synthesized type name for the provided anonymous type or nil
// if the type is not registered.
func (at *AnonymousTypes) Get(t types.Type) *types.TypeName {
	s, _ := at.m.At(t).(*types.TypeName)
	return s
}

// Ordered returns synthesized type names for the registered anonymous types in
// the order they were registered.
func (at *AnonymousTypes) Ordered() []*types.TypeName {
	return at.order
}

// Register a synthesized type name for an anonymous type.
func (at *AnonymousTypes) Register(name *types.TypeName, anonType types.Type) {
	at.m.Set(anonType, name)
	at.order = append(at.order, name)
}

// IsGeneric returns true if the provided type is a type parameter or depends
// on a type parameter.
func IsGeneric(t types.Type) bool {
	switch t := t.(type) {
	case *types.Array:
		return IsGeneric(t.Elem())
	case *types.Basic:
		return false
	case *types.Chan:
		return IsGeneric(t.Elem())
	case *types.Interface:
		for i := 0; i < t.NumMethods(); i++ {
			if IsGeneric(t.Method(i).Type()) {
				return true
			}
		}
		for i := 0; i < t.NumEmbeddeds(); i++ {
			if IsGeneric(t.Embedded(i)) {
				return true
			}
		}
		return false
	case *types.Map:
		return IsGeneric(t.Key()) || IsGeneric(t.Elem())
	case *types.Named:
		for i := 0; i < t.TypeArgs().Len(); i++ {
			if IsGeneric(t.TypeArgs().At(i)) {
				return true
			}
		}
		return false
	case *types.Pointer:
		return IsGeneric(t.Elem())
	case *types.Slice:
		return IsGeneric(t.Elem())
	case *types.Signature:
		for i := 0; i < t.Params().Len(); i++ {
			if IsGeneric(t.Params().At(i).Type()) {
				return true
			}
		}
		for i := 0; i < t.Results().Len(); i++ {
			if IsGeneric(t.Results().At(i).Type()) {
				return true
			}
		}
		return false
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			if IsGeneric(t.Field(i).Type()) {
				return true
			}
		}
		return false
	case *types.TypeParam:
		return true
	default:
		panic(fmt.Errorf("%v has unexpected type %T", t, t))
	}
}

// IsMethod returns true if the passed object is a method.
func IsMethod(o types.Object) bool {
	f, ok := o.(*types.Func)
	return ok && f.Type().(*types.Signature).Recv() != nil
}

// TypeParams returns a list of type parameters for a parameterized type, or
// nil otherwise.
func TypeParams(t types.Type) *types.TypeParamList {
	named, ok := t.(*types.Named)
	if !ok {
		return nil
	}
	return named.TypeParams()
}
