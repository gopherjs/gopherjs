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

// RecvAsFirstArg takes a method signature and returns a function
// signature with receiver as the first parameter.
func RecvAsFirstArg(sig *types.Signature) *types.Signature {
	params := make([]*types.Var, 0, 1+sig.Params().Len())
	params = append(params, sig.Recv())
	for i := 0; i < sig.Params().Len(); i++ {
		params = append(params, sig.Params().At(i))
	}
	return types.NewSignatureType(nil, nil, nil, types.NewTuple(params...), sig.Results(), sig.Variadic())
}

// Selection is a common interface for go/types.Selection and our custom-constructed
// method and field selections.
type Selection interface {
	Kind() types.SelectionKind
	Recv() types.Type
	Index() []int
	Obj() types.Object
	Type() types.Type
}

// NewSelection creates a new selection.
func NewSelection(kind types.SelectionKind, recv types.Type, index []int, obj types.Object, typ types.Type) Selection {
	return &selectionImpl{
		kind:  kind,
		recv:  recv,
		index: index,
		obj:   obj,
		typ:   typ,
	}
}

type selectionImpl struct {
	kind  types.SelectionKind
	recv  types.Type
	index []int
	obj   types.Object
	typ   types.Type
}

func (sel *selectionImpl) Kind() types.SelectionKind { return sel.kind }
func (sel *selectionImpl) Recv() types.Type          { return sel.recv }
func (sel *selectionImpl) Index() []int              { return sel.index }
func (sel *selectionImpl) Obj() types.Object         { return sel.obj }
func (sel *selectionImpl) Type() types.Type          { return sel.typ }
