//go:build js

package reflect

import (
	"strconv"
	"unsafe"

	"internal/abi"

	"github.com/gopherjs/gopherjs/js"
)

func init() {
	// avoid dead code elimination
	used := func(i any) {}
	used(rtype{})
	used(uncommonType{})
	used(arrayType{})
	used(chanType{})
	used(funcType{})
	used(interfaceType{})
	used(mapType{})
	used(ptrType{})
	used(sliceType{})
	used(structType{})
	used(structField{})
	used(toKindTypeExt)
}

//gopherjs:new
func toAbiType(typ Type) *abi.Type {
	return typ.(*rtype).common()
}

//gopherjs:new
func jsType(typ Type) *js.Object {
	return toAbiType(typ).JsType()
}

//gopherjs:replace
func (t *rtype) ptrTo() *abi.Type {
	return abi.ReflectType(js.Global.Call("$ptrType", jsType(t)))
}

//gopherjs:replace
func toRType(t *abi.Type) *rtype {
	rtyp := &rtype{}
	// Assign t to the abiType. The abiType is a `*Type` and the t
	// field on `rtype` is `Type`. However, this is valid because of how
	// pointers and references work in JS. We set this so that the t
	// isn't a copy but the actual abiType object.
	js.InternalObject(rtyp).Set("t", js.InternalObject(t))
	return rtyp
}

//gopherjs:replace
func (t *rtype) String() string {
	return toAbiType(t).String()
}

//gopherjs:purge
func addReflectOff(ptr unsafe.Pointer) int32

//gopherjs:replace
func (t *rtype) nameOff(off aNameOff) abi.Name {
	return toAbiType(t).NameOff(off)
}

//gopherjs:replace
func resolveReflectName(n abi.Name) aNameOff {
	return abi.ResolveReflectName(n)
}

//gopherjs:replace
func (t *rtype) typeOff(off aTypeOff) *abi.Type {
	return toAbiType(t).TypeOff(off)
}

//gopherjs:replace
func resolveReflectType(t *abi.Type) aTypeOff {
	return abi.ResolveReflectType(t)
}

//gopherjs:replace
func (t *rtype) textOff(off aTextOff) unsafe.Pointer {
	return toAbiType(t).TextOff(off)
}

//gopherjs:replace
func resolveReflectText(ptr unsafe.Pointer) aTextOff {
	return abi.ResolveReflectText(ptr)
}

//gopherjd:replace
func pkgPath(n abi.Name) string {
	return n.PkgPath()
}

//gopherjs:replace
func TypeOf(i any) Type {
	if i == nil {
		return nil
	}
	return toRType(rtypeOf(i))
}

//gopherjs:replace
func rtypeOf(i any) *abi.Type {
	return abi.ReflectType(js.InternalObject(i).Get("constructor"))
}

//gopherjs:purge Unused type
type common struct{}

//gopherjs:replace
func ArrayOf(count int, elem Type) Type {
	if count < 0 {
		panic("reflect: negative length passed to ArrayOf")
	}

	return toRType(abi.ReflectType(js.Global.Call("$arrayType", jsType(elem), count)))
}

//gopherjs:replace
func ChanOf(dir ChanDir, t Type) Type {
	return toRType(abi.ReflectType(js.Global.Call("$chanType", jsType(t), dir == SendDir, dir == RecvDir)))
}

//gopherjs:replace
func FuncOf(in, out []Type, variadic bool) Type {
	if variadic && (len(in) == 0 || in[len(in)-1].Kind() != Slice) {
		panic("reflect.FuncOf: last arg of variadic func must be slice")
	}

	jsIn := make([]*js.Object, len(in))
	for i, v := range in {
		jsIn[i] = jsType(v)
	}
	jsOut := make([]*js.Object, len(out))
	for i, v := range out {
		jsOut[i] = jsType(v)
	}
	return toRType(abi.ReflectType(js.Global.Call("$funcType", jsIn, jsOut, variadic)))
}

//gopherjs:replace
func MapOf(key, elem Type) Type {
	switch key.Kind() {
	case Func, Map, Slice:
		panic("reflect.MapOf: invalid key type " + key.String())
	}

	return toRType(abi.ReflectType(js.Global.Call("$mapType", jsType(key), jsType(elem))))
}

//gopherjs:replace
func SliceOf(t Type) Type {
	return toRType(abi.ReflectType(js.Global.Call("$sliceType", jsType(t))))
}

//gopherjs:replace
func StructOf(fields []StructField) Type {
	var (
		jsFields  = make([]*js.Object, len(fields))
		fset      = map[string]struct{}{}
		pkgpath   string
		hasGCProg bool
	)
	for i, field := range fields {
		if field.Name == "" {
			panic("reflect.StructOf: field " + strconv.Itoa(i) + " has no name")
		}
		if !isValidFieldName(field.Name) {
			panic("reflect.StructOf: field " + strconv.Itoa(i) + " has invalid name")
		}
		if field.Type == nil {
			panic("reflect.StructOf: field " + strconv.Itoa(i) + " has no type")
		}
		f, fpkgpath := runtimeStructField(field)
		ft := f.Typ
		if ft.Kind()&kindGCProg != 0 {
			hasGCProg = true
		}
		if fpkgpath != "" {
			if pkgpath == "" {
				pkgpath = fpkgpath
			} else if pkgpath != fpkgpath {
				panic("reflect.Struct: fields with different PkgPath " + pkgpath + " and " + fpkgpath)
			}
		}
		name := field.Name
		if f.Embedded() {
			// Embedded field
			if field.Type.Kind() == Ptr {
				// Embedded ** and *interface{} are illegal
				elem := field.Type.Elem()
				if k := elem.Kind(); k == Ptr || k == Interface {
					panic("reflect.StructOf: illegal anonymous field type " + field.Type.String())
				}
			}
			switch field.Type.Kind() {
			case Interface:
			case Ptr:
				ptr := (*ptrType)(unsafe.Pointer(ft))
				if unt := ptr.Uncommon(); unt != nil {
					if i > 0 && unt.Mcount > 0 {
						// Issue 15924.
						panic("reflect: embedded type with methods not implemented if type is not first field")
					}
					if len(fields) > 1 {
						panic("reflect: embedded type with methods not implemented if there is more than one field")
					}
				}
			default:
				if unt := ft.Uncommon(); unt != nil {
					if i > 0 && unt.Mcount > 0 {
						// Issue 15924.
						panic("reflect: embedded type with methods not implemented if type is not first field")
					}
					if len(fields) > 1 && ft.Kind()&kindDirectIface != 0 {
						panic("reflect: embedded type with methods not implemented for non-pointer type")
					}
				}
			}
		}

		if _, dup := fset[name]; dup && name != "_" {
			panic("reflect.StructOf: duplicate field " + name)
		}
		fset[name] = struct{}{}
		// To be consistent with Compiler's behavior we need to avoid externalizing
		// the "name" property. The line below is effectively an inverse of the
		// internalStr() function.
		jsf := js.InternalObject(struct{ name string }{name})
		// The rest is set through the js.Object() interface, which the compiler will
		// externalize for us.
		jsf.Set("prop", name)
		jsf.Set("exported", f.Name.IsExported())
		jsf.Set("typ", jsType(field.Type))
		jsf.Set("tag", field.Tag)
		jsf.Set("embedded", field.Anonymous)
		jsFields[i] = jsf
	}
	_ = hasGCProg
	typ := js.Global.Call("$structType", "", jsFields)
	if pkgpath != "" {
		typ.Set("pkgPath", pkgpath)
	}
	return toRType(abi.ReflectType(typ))
}

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
