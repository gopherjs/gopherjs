package typesutil

import (
	"fmt"
	"go/types"
	"os"
	"strings"
)

func IsJsPackage(pkg *types.Package) bool {
	// TODO: Resolve issue #415 and remove this temporary workaround.
	if pkg != nil && strings.HasSuffix(pkg.Path(), "/vendor/github.com/gopherjs/gopherjs/js") {
		fmt.Fprintln(os.Stderr, "GopherJS: vendoring github.com/gopherjs/gopherjs/js package is not supported, see https://github.com/gopherjs/gopherjs/issues/415.")
		os.Exit(1)
	}

	return pkg != nil && (pkg.Path() == "github.com/gopherjs/gopherjs/js" || strings.HasSuffix(pkg.Path(), "/vendor/github.com/gopherjs/gopherjs/js"))
}

func IsJsObject(t types.Type) bool {
	ptr, isPtr := t.(*types.Pointer)
	if !isPtr {
		return false
	}
	named, isNamed := ptr.Elem().(*types.Named)
	return isNamed && IsJsPackage(named.Obj().Pkg()) && named.Obj().Name() == "Object"
}
