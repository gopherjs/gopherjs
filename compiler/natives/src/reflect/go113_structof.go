// +build js
// +build !go1.14

package reflect

import (
	"strconv"
	"unsafe"

	"github.com/gopherjs/gopherjs/js"
)

func StructOf(fields []StructField) Type {
	var (
		jsFields  = make([]*js.Object, len(fields))
		fset      = map[string]struct{}{}
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
		f := runtimeStructField(field)
		ft := f.typ
		if ft.kind&kindGCProg != 0 {
			hasGCProg = true
		}

		name := field.Name
		if f.embedded() {
			// Embedded field
			if f.typ.Kind() == Ptr {
				// Embedded ** and *interface{} are illegal
				elem := ft.Elem()
				if k := elem.Kind(); k == Ptr || k == Interface {
					panic("reflect.StructOf: illegal embedded field type " + ft.String())
				}
			}

			switch f.typ.Kind() {
			case Interface:
				ift := (*interfaceType)(unsafe.Pointer(ft))
				for _, m := range ift.methods {
					if ift.nameOff(m.name).pkgPath() != "" {
						// TODO(sbinet).  Issue 15924.
						panic("reflect: embedded interface with unexported method(s) not implemented")
					}
				}
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
					for _, m := range unt.methods() {
						mname := ptr.nameOff(m.name)
						if mname.pkgPath() != "" {
							// TODO(sbinet).
							// Issue 15924.
							panic("reflect: embedded interface with unexported method(s) not implemented")
						}
					}
				}
				if unt := ptr.elem.uncommon(); unt != nil {
					for _, m := range unt.methods() {
						mname := ptr.nameOff(m.name)
						if mname.pkgPath() != "" {
							// TODO(sbinet)
							// Issue 15924.
							panic("reflect: embedded interface with unexported method(s) not implemented")
						}
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
					for _, m := range unt.methods() {
						mname := ft.nameOff(m.name)
						if mname.pkgPath() != "" {
							// TODO(sbinet)
							// Issue 15924.
							panic("reflect: embedded interface with unexported method(s) not implemented")
						}
					}
				}
			}
		}
		if _, dup := fset[name]; dup {
			panic("reflect.StructOf: duplicate field " + name)
		}
		fset[name] = struct{}{}
		jsf := js.Global.Get("Object").New()
		jsf.Set("prop", name)
		jsf.Set("name", name)
		jsf.Set("exported", f.name.isExported())
		jsf.Set("typ", jsType(field.Type))
		jsf.Set("tag", field.Tag)
		jsf.Set("embedded", field.Anonymous)
		jsFields[i] = jsf
	}
	_ = hasGCProg
	typ := js.Global.Call("$structType", "", jsFields)
	return _reflectType(typ, objstr)
}

func objstr(obj *js.Object) string {
	return obj.String()
}
