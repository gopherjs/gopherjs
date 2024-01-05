package typesutil

import "go/types"

func IsJsPackage(pkg *types.Package) bool {
	return pkg != nil && pkg.Path() == "github.com/gopherjs/gopherjs/js"
}

func IsJsObject(t types.Type) bool {
	ptr, isPtr := t.(*types.Pointer)
	if !isPtr {
		return false
	}
	named, isNamed := ptr.Elem().(*types.Named)
	return isNamed && IsJsPackage(named.Obj().Pkg()) && named.Obj().Name() == "Object"
}

// RecvType returns a named type of a method receiver, or nil if it's not a method.
//
// For methods on a pointer receiver, the underlying named type is returned.
func RecvType(sig *types.Signature) *types.Named {
	recv := sig.Recv()
	if recv == nil {
		return nil
	}

	typ := recv.Type()
	if ptrType, ok := typ.(*types.Pointer); ok {
		typ = ptrType.Elem()
	}

	return typ.(*types.Named)
}
