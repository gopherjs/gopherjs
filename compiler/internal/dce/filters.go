package dce

import (
	"go/types"

	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

// getFilters determines the DCE filters for the given object.
// This will return an object filter and optionally return a method filter.
func getFilters(o types.Object) (objectFilter, methodFilter string) {
	importPath := o.Pkg().Path()
	if typesutil.IsMethod(o) {
		recv := typesutil.RecvType(o.Type().(*types.Signature)).Obj()
		objectFilter = importPath + `.` + recv.Name()
		if !o.Exported() {
			methodFilter = importPath + `.` + o.Name() + `~`
		}
	} else {
		objectFilter = importPath + `.` + o.Name()
	}
	return
}

// getDepFilter returns the filter for the given object to be used as a dependency.
func getDepFilter(o types.Object) string {
	qualifiedName := o.Pkg().Path() + "." + o.Name()
	if typesutil.IsMethod(o) {
		qualifiedName += "~"
	}
	return qualifiedName
}
