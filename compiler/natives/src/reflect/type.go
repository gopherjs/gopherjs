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
}

// GOPHERJS: In Go the rtype and ABI Type share a memory footprint so a pointer
// to one is a pointer to the other. This doesn't work in JS so instead we construct
// a new rtype with a `*abi.Type` inside of it. However, that means the pointers are
// different and multiple `*rtypes` may point to the same ABI type, so we have to
// take that into account when overriding code or leaving the original code.
// (reflectlite does this better by having the rtype not be a pointer.)
//
//gopherjs:replace
type rtype struct {
	t *abi.Type
}

//gopherjs:replace
func (t *rtype) common() *abi.Type {
	return t.t
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
	return toAbiType(t).PtrTo()
}

//gopherjs:replace
func toRType(t *abi.Type) *rtype {
	return &rtype{t: t}
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

//gopherjs:replace Same issue as rtype
type interfaceType struct {
	*abi.InterfaceType
}

//gopherjs:new
func toInterfaceType(typ *abi.Type) *interfaceType {
	if tt := typ.InterfaceType(); tt != nil {
		return &interfaceType{InterfaceType: tt}
	}
	return nil
}

//gopherjs:replace Same issue as rtype
type mapType struct {
	*abi.MapType
}

//gopherjs:new
func toMapType(typ *abi.Type) *mapType {
	if tt := typ.MapType(); tt != nil {
		return &mapType{MapType: tt}
	}
	return nil
}

//gopherjs:replace Same issue as rtype
type ptrType struct {
	*abi.PtrType
}

//gopherjs:new
func toPtrType(typ *abi.Type) *ptrType {
	if tt := typ.PtrType(); tt != nil {
		return &ptrType{PtrType: tt}
	}
	return nil
}

//gopherjs:replace Same issue as rtype
type sliceType struct {
	*abi.SliceType
}

//gopherjs:new
func toSliceType(typ *abi.Type) *sliceType {
	if tt := typ.SliceType(); tt != nil {
		return &sliceType{SliceType: tt}
	}
	return nil
}

//gopherjs:replace Same issue as rtype
type structType struct {
	*abi.StructType
}

//gopherjs:new
func toStructType(typ *abi.Type) *structType {
	if tt := typ.StructType(); tt != nil {
		return &structType{StructType: tt}
	}
	return nil
}

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
				ptr := toPtrType(ft)
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

//gopherjs:replace Used pointer cast to interface kind type.
func (t *rtype) NumMethod() int {
	if tt := toInterfaceType(t.common()); tt != nil {
		return tt.NumMethod()
	}
	return len(t.exportedMethods())
}

//gopherjs:replace Used pointer cast to struct kind type.
func (t *rtype) Field(i int) StructField {
	if tt := toStructType(t.common()); tt != nil {
		return tt.Field(i)
	}
	panic("reflect: Field of non-struct type " + t.String())
}

//gopherjs:replace Used pointer cast to struct kind type.
func (t *rtype) FieldByIndex(index []int) StructField {
	if tt := toStructType(t.common()); tt != nil {
		return tt.FieldByIndex(index)
	}
	panic("reflect: FieldByIndex of non-struct type " + t.String())
}

//gopherjs:replace Used pointer cast to struct kind type.
func (t *rtype) FieldByName(name string) (StructField, bool) {
	if tt := toStructType(t.common()); tt != nil {
		return tt.FieldByName(name)
	}
	panic("reflect: FieldByName of non-struct type " + t.String())
}

//gopherjs:replace Used pointer cast to struct kind type.
func (t *rtype) FieldByNameFunc(match func(string) bool) (StructField, bool) {
	if tt := toStructType(t.common()); tt != nil {
		return tt.FieldByNameFunc(match)
	}
	panic("reflect: FieldByNameFunc of non-struct type " + t.String())
}

//gopherjs:replace Used pointer cast to map kind type.
func (t *rtype) Key() Type {
	if tt := toMapType(t.common()); tt != nil {
		return toType(tt.Key)
	}
	panic("reflect: Key of non-map type " + t.String())
}

//gopherjs:replace Used pointer cast to array kind type.
func (t *rtype) Len() int {
	if tt := t.common().ArrayType(); tt != nil {
		return int(tt.Len)
	}
	panic("reflect: Len of non-array type " + t.String())
}

//gopherjs:replace Used pointer cast to struct kind type.
func (t *rtype) NumField() int {
	if tt := toStructType(t.common()); tt != nil {
		return len(tt.Fields)
	}
	panic("reflect: NumField of non-struct type " + t.String())
}

//gopherjs:replace Used pointer cast to func kind type.
func (t *rtype) In(i int) Type {
	if tt := t.common().FuncType(); tt != nil {
		return toType(tt.InSlice()[i])
	}
	panic("reflect: In of non-func type " + t.String())
}

//gopherjs:replace Used pointer cast to fun kind type.
func (t *rtype) NumIn() int {
	if tt := t.common().FuncType(); tt != nil {
		return tt.NumIn()
	}
	panic("reflect: NumIn of non-func type " + t.String())
}

//gopherjs:replace Used pointer cast to func kind type.
func (t *rtype) NumOut() int {
	if tt := t.common().FuncType(); tt != nil {
		return tt.NumOut()
	}
	panic("reflect: NumOut of non-func type " + t.String())
}

//gopherjs:replace Used pointer cast to func kind type.
func (t *rtype) Out(i int) Type {
	if tt := t.common().FuncType(); tt != nil {
		return toType(tt.OutSlice()[i])
	}
	panic("reflect: Out of non-func type " + t.String())
}

//gopherjs:replace Used pointer cast to func kind type.
func (t *rtype) IsVariadic() bool {
	if tt := t.common().FuncType(); tt != nil {
		return tt.IsVariadic()
	}
	panic("reflect: IsVariadic of non-func type " + t.String())
}
