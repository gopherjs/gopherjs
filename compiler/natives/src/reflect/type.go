//go:build js

package reflect

import (
	"internal/abi"

	"github.com/gopherjs/gopherjs/js"
)

//gopherjs:replace
func (t *rtype) String() string {
	return toAbiType(t).String()
}

//gopherjs:replace
func rtypeOf(i any) *abi.Type {
	return abi.ReflectType(js.InternalObject(i).Get("constructor"))
}

//gopherjs:purge Unused type
type common struct{}

//gopherjs:purge Used in original MapOf and not used in override MapOf by GopherJS
func bucketOf(ktyp, etyp *abi.Type) *abi.Type

//gopherjs:purge Relates to GC programs not valid for GopherJS
func (t *rtype) gcSlice(begin, end uintptr) []byte

//gopherjs:purge Relates to GC programs not valid for GopherJS
func emitGCMask(out []byte, base uintptr, typ *abi.Type, n uintptr)

//gopherjs:purge Relates to GC programs not valid for GopherJS
func appendGCProg(dst []byte, typ *abi.Type) []byte

// toKindTypeExt will be automatically called when a cast to one of the
// extended kind types is performed.
//
// This is similar to `kindType` except that the reflect package has
// extended several of the kind types to have additional methods added to them.
// To get access to those methods, the `kindTypeExt` is checked or created.
// The automatic cast is handled in compiler/expressions.go
//
// gopherjs:new
func toKindTypeExt(src any) *js.Object {
	var abiTyp *abi.Type
	switch t := src.(type) {
	case *rtype:
		abiTyp = t.common()
	case Type:
		abiTyp = toAbiType(t)
	case *abi.Type:
		abiTyp = t
	default:
		panic(`unexpected type in toKindTypeExt`)
	}

	const (
		idKindType    = `kindType`
		idKindTypeExt = `kindTypeExt`
	)
	// Check if a kindTypeExt has already been created for this type.
	ext := js.InternalObject(abiTyp).Get(idKindTypeExt)
	if ext != js.Undefined {
		return ext
	}

	// Constructe a new kindTypeExt for this type.
	kindType := js.InternalObject(abiTyp).Get(idKindType)
	switch abiTyp.Kind() {
	case abi.Interface:
		ext = js.InternalObject(&interfaceType{})
		ext.Set(`InterfaceType`, js.InternalObject(kindType))
	case abi.Map:
		ext = js.InternalObject(&mapType{})
		ext.Set(`MapType`, js.InternalObject(kindType))
	case abi.Pointer:
		ext = js.InternalObject(&ptrType{})
		ext.Set(`PtrType`, js.InternalObject(kindType))
	case abi.Slice:
		ext = js.InternalObject(&sliceType{})
		ext.Set(`SliceType`, js.InternalObject(kindType))
	case abi.Struct:
		ext = js.InternalObject(&structType{})
		ext.Set(`StructType`, js.InternalObject(kindType))
	default:
		panic(`unexpected kind in toKindTypeExt`)
	}
	js.InternalObject(abiTyp).Set(idKindTypeExt, ext)
	return ext
}
