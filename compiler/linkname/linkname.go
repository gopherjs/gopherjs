package linkname

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/gopherjs/gopherjs/compiler/astutil"
	"github.com/gopherjs/gopherjs/compiler/internal/symbol"
	"github.com/gopherjs/gopherjs/internal/errorList"
)

// GoLinkname describes a go:linkname compiler directive found in the source code.
//
// GopherJS treats these directives in a way that resembles a symbolic link,
// where for a single given symbol implementation there may be zero or more
// symbols referencing it. This is subtly different from the upstream Go
// implementation, which simply overrides symbol name the linker will use.
type GoLinkname struct {
	Implementation symbol.Name
	Reference      symbol.Name
}

// ParseGoLinknames processed comments in a source file and extracts //go:linkname
// compiler directive from the comments.
//
// The following directive format is supported:
// //go:linkname <localname> <importpath>.<name>
// //go:linkname <localname> <importpath>.<type>.<name>
// //go:linkname <localname> <importpath>.<(*type)>.<name>
//
// GopherJS directive support has the following limitations:
//
//   - External linkname must be specified.
//   - The directive must be applied to a package-level function or method (variables
//     are not supported).
//   - The local function referenced by the directive must have no body (in other
//     words, it can only "import" an external function implementation into the
//     local scope).
func ParseGoLinknames(fset *token.FileSet, pkgPath string, file *ast.File) ([]GoLinkname, error) {
	var errs errorList.ErrorList = nil
	var directives []GoLinkname

	isUnsafe := astutil.ImportsUnsafe(file)

	processComment := func(comment *ast.Comment) error {
		link, err := readLinknameFromComment(pkgPath, comment)
		if err != nil || link == nil {
			return err
		}

		if !isUnsafe {
			return fmt.Errorf(`//go:linkname is only allowed in Go files that import "unsafe"`)
		}

		obj := file.Scope.Lookup(link.Reference.Name)
		if obj == nil {
			return fmt.Errorf("//go:linkname local symbol %q is not found in the current source file", link.Reference.Name)
		}

		if obj.Kind != ast.Fun {
			if isMitigatedVarLinkname(link.Reference) {
				return nil
			}
			return fmt.Errorf("gopherjs: //go:linkname is only supported for functions, got %q", obj.Kind)
		}

		if decl := obj.Decl.(*ast.FuncDecl); decl.Body != nil {
			if isMitigatedInsertLinkname(link.Reference) {
				return nil
			}
			return fmt.Errorf("gopherjs: //go:linkname can not insert local implementation into an external package %q", link.Implementation.PkgPath)
		}

		// Local function has no body, treat it as a reference to an external implementation.
		directives = append(directives, *link)
		return nil
	}

	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if err := processComment(c); err != nil {
				errs = append(errs, errorAt(err, fset, c.Pos()))
			}
		}
	}

	return directives, errs.ErrOrNil()
}

// errorAt annotates an error with a position in the source code.
func errorAt(err error, fset *token.FileSet, pos token.Pos) error {
	return fmt.Errorf("%s: %w", fset.Position(pos), err)
}

// GoLinknameSet is a utility that enables quick lookup of whether a decl is
// affected by any go:linkname directive in the program.
type GoLinknameSet struct {
	byImplementation map[symbol.Name][]GoLinkname
	byReference      map[symbol.Name]GoLinkname
}

// Add more GoLinkname directives into the set.
func (gls *GoLinknameSet) Add(entries []GoLinkname) error {
	if gls.byImplementation == nil {
		gls.byImplementation = map[symbol.Name][]GoLinkname{}
	}
	if gls.byReference == nil {
		gls.byReference = map[symbol.Name]GoLinkname{}
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
func (gls *GoLinknameSet) IsImplementation(sym symbol.Name) bool {
	_, found := gls.byImplementation[sym]
	return found
}

// FindImplementation returns a symbol name, which provides the implementation
// for the given symbol. The second value indicates whether the implementation
// was found.
func (gls *GoLinknameSet) FindImplementation(sym symbol.Name) (symbol.Name, bool) {
	directive, found := gls.byReference[sym]
	return directive.Implementation, found
}
