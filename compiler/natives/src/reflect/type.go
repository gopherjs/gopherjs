//go:build js

package reflect

import (
	"strconv"
	"unsafe"

	"internal/abi"

	"github.com/gopherjs/gopherjs/js"
)

var initialized = false

func init() {
	// avoid dead code elimination
	used := func(i any) {}
	used(rtype{})
	used(uncommonType{})
	used(method{})
	used(arrayType{})
	used(chanType{})
	used(funcType{})
	used(interfaceType{})
	used(mapType{})
	used(ptrType{})
	used(sliceType{})
	used(structType{})
	used(imethod{})
	used(structField{})

	initialized = true
	uint8Type = TypeOf(uint8(0)).(*rtype) // set for real
}

//gopherjs:new
func toAbiType(typ Type) *abi.Type {
	return typ.(*rtype).common()
}

//gopherjs:new
func jsType(typ Type) *js.Object {
	return toAbiType(typ).JsType()
}

func (t *rtype) ptrTo() *abi.Type {
	return toAbiType(t).PtrTo()
}

//gopherjs:purge
func addReflectOff(ptr unsafe.Pointer) int32

//gopherjs:replace
func (t *rtype) nameOff(off aNameOff) abi.Name {
	return t.NameOff(off)
}

//gopherjs:replace
func resolveReflectName(n abi.Name) aNameOff {
	return abi.ResolveReflectName(n)
}

//gopherjs:replace
func (t *rtype) typeOff(off aTypeOff) *abi.Type {
	return t.TypeOff(off)
}

//gopherjs:replace
func resolveReflectType(t *abi.Type) aTypeOff {
	return abi.ResolveReflectType(t)
}

//gopherjs:replace
func (t *rtype) textOff(off aTextOff) unsafe.Pointer {
	return t.TextOff(off)
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
	if !initialized { // avoid error of uint8Type
		return &rtype{}
	}
	if i == nil {
		return nil
	}
	return reflectType(js.InternalObject(i).Get("constructor"))
}

//gopherjs:replace
func rtypeOf(i any) *abi.Type {
	return abi.ReflectType(js.InternalObject(i).Get("constructor"))
}

func ArrayOf(count int, elem Type) Type {
	if count < 0 {
		panic("reflect: negative length passed to ArrayOf")
	}

	return reflectType(js.Global.Call("$arrayType", jsType(elem), count))
}

func ChanOf(dir ChanDir, t Type) Type {
	return reflectType(js.Global.Call("$chanType", jsType(t), dir == SendDir, dir == RecvDir))
}

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
	return reflectType(js.Global.Call("$funcType", jsIn, jsOut, variadic))
}

func MapOf(key, elem Type) Type {
	switch key.Kind() {
	case Func, Map, Slice:
		panic("reflect.MapOf: invalid key type " + key.String())
	}

	return reflectType(js.Global.Call("$mapType", jsType(key), jsType(elem)))
}

func SliceOf(t Type) Type {
	return reflectType(js.Global.Call("$sliceType", jsType(t)))
}

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
		ft := f.typ
		if ft.kind&kindGCProg != 0 {
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
		if f.embedded() {
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
				if unt := ptr.uncommon(); unt != nil {
					if i > 0 && unt.mcount > 0 {
						// Issue 15924.
						panic("reflect: embedded type with methods not implemented if type is not first field")
					}
					if len(fields) > 1 {
						panic("reflect: embedded type with methods not implemented if there is more than one field")
					}
				}
			default:
				if unt := ft.uncommon(); unt != nil {
					if i > 0 && unt.mcount > 0 {
						// Issue 15924.
						panic("reflect: embedded type with methods not implemented if type is not first field")
					}
					if len(fields) > 1 && ft.kind&kindDirectIface != 0 {
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
		jsf.Set("exported", f.name.isExported())
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
	return reflectType(typ)
}
