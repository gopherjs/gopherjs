package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/astutil"
)

// GoLinkname describes a go:linkname compiler directive found in the source code.
//
// GopherJS treats these directives in a way that resembles a symbolic link,
// where for a single given symbol implementation there may be zero or more
// symbols referencing it. This is subtly different from the upstream Go
// implementation, which simply overrides symbol name the linker will use.
type GoLinkname struct {
	Implementation SymName
	Reference      SymName
}

// SymName uniquely identifies a named submol within a program.
//
// This is a logical equivalent of a symbol name used by traditional linkers.
// The following properties should hold true:
//
//  - Each named symbol within a program has a unique SymName.
//  - Similarly named methods of different types will have different symbol names.
//  - The string representation is opaque and should not be attempted to reversed
//    to a struct form.
type SymName struct {
	PkgPath string // Full package import path.
	Name    string // Symbol name.
}

// newSymName constructs SymName for a given named symbol.
func newSymName(o types.Object) SymName {
	if fun, ok := o.(*types.Func); ok {
		sig := fun.Type().(*types.Signature)
		if recv := sig.Recv(); recv != nil {
			// Special case: disambiguate names for different types' methods.
			return SymName{
				PkgPath: o.Pkg().Path(),
				Name:    recv.Type().(*types.Named).Obj().Name() + "." + o.Name(),
			}
		}
	}
	return SymName{
		PkgPath: o.Pkg().Path(),
		Name:    o.Name(),
	}
}

func (n SymName) String() string { return n.PkgPath + "." + n.Name }

// parseGoLinknames processed comments in a source file and extracts //go:linkname
// compiler directive from the comments.
//
// The following directive format is supported:
// //go:linkname <localname> <importpath>.<name>
//
// GopherJS directive support has the following limitations:
//
//  - External linkname must be specified.
//  - The directive must be applied to a package-level function (variables and
//    methods are not supported).
//  - The local function referenced by the directive must have no body (in other
//    words, it can only "import" an external function implementation into the
//    local scope).
func parseGoLinknames(fset *token.FileSet, pkgPath string, file *ast.File) ([]GoLinkname, error) {
	var errs ErrorList = nil
	var directives []GoLinkname

	isUnsafe := astutil.ImportsUnsafe(file)

	processComment := func(comment *ast.Comment) error {
		if !strings.HasPrefix(comment.Text, "//go:linkname ") {
			return nil // Not a linkname compiler directive.
		}

		// TODO(nevkontakte): Ideally we should check that the directive comment
		// is on a line by itself, line Go compiler does, but ast.Comment doesn't
		// provide an easy way to find that out.

		if !isUnsafe {
			return fmt.Errorf(`//go:linkname is only allowed in Go files that import "unsafe"`)
		}

		fields := strings.Fields(comment.Text)
		if len(fields) != 3 {
			return fmt.Errorf(`usage (all fields required): //go:linkname localname importpath.extname`)
		}

		localPkg, localName := pkgPath, fields[1]
		extPkg, extName := "", fields[2]
		if idx := strings.LastIndexByte(extName, '.'); idx != -1 {
			extPkg, extName = extName[0:idx], extName[idx+1:]
		}

		obj := file.Scope.Lookup(localName)
		if obj == nil {
			if pkgPath == "syscall" {
				// Syscall uses go:cgo_import_dynamic pragma to import symbols from
				// dynamic libraries when build with GOOS=darwin, which GopherJS doesn't
				// support. Silently ignore such directives.
				//
				// In the long term https://github.com/gopherjs/gopherjs/issues/693 is a
				// preferred solution.
				return nil
			}
			return fmt.Errorf("//go:linkname local symbol %q is not found in the current source file", localName)
		}

		if obj.Kind != ast.Fun {
			if pkgPath == "math/bits" || pkgPath == "reflect" {
				// These standard library packages are known to use go:linkname with
				// variables, which GopherJS doesn't support. We silently ignore such
				// directives, since it doesn't seem to cause any problems.
				return nil
			}
			return fmt.Errorf("gopherjs: //go:linkname is only supported for functions, got %q", obj.Kind)
		}

		decl := obj.Decl.(*ast.FuncDecl)
		if decl.Body != nil {
			if pkgPath == "runtime" || pkgPath == "internal/bytealg" {
				// These standard library packages are known to use unsupported
				// "insert"-style go:linkname directives, which we ignore here and handle
				// case-by-case in native overrides.
				return nil
			}
			return fmt.Errorf("gopherjs: //go:linkname can not insert local implementation into an external package %q", extPkg)
		}
		// Local function has no body, treat it as a reference to an external implementation.
		directives = append(directives, GoLinkname{
			Reference:      SymName{PkgPath: localPkg, Name: localName},
			Implementation: SymName{PkgPath: extPkg, Name: extName},
		})
		return nil
	}

	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if err := processComment(c); err != nil {
				errs = append(errs, ErrorAt(err, fset, c.Pos()))
			}
		}
	}

	return directives, errs.Normalize()
}

// goLinknameSet is a utility that enables quick lookup of whether a decl is
// affected by any go:linkname directive in the program.
type goLinknameSet struct {
	byImplementation map[SymName][]GoLinkname
	byReference      map[SymName]GoLinkname
}

// Add more GoLinkname directives into the set.
func (gls *goLinknameSet) Add(entries []GoLinkname) error {
	if gls.byImplementation == nil {
		gls.byImplementation = map[SymName][]GoLinkname{}
	}
	if gls.byReference == nil {
		gls.byReference = map[SymName]GoLinkname{}
	}
	for _, e := range entries {
		gls.byImplementation[e.Implementation] = append(gls.byImplementation[e.Implementation], e)
		if prev, found := gls.byReference[e.Reference]; found {
			return fmt.Errorf("conflicting go:linkname directives: two implementations for %q: %q and %q",
				e.Reference, prev.Implementation, e.Implementation)
		}
		gls.byReference[e.Reference] = e
	}
	return nil
}

// IsImplementation returns true if there is a directive referencing this symbol
// as an implementation.
func (gls *goLinknameSet) IsImplementation(sym SymName) bool {
	_, found := gls.byImplementation[sym]
	return found
}

// FindImplementation returns a symbol name, which provides the implementation
// for the given symbol. The second value indicates whether the implementation
// was found.
func (gls *goLinknameSet) FindImplementation(sym SymName) (SymName, bool) {
	directive, found := gls.byReference[sym]
	return directive.Implementation, found
}
