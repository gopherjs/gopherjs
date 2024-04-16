package compiler

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/astutil"
	"github.com/gopherjs/gopherjs/compiler/internal/symbol"
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

// readLinknameFromComment reads the given comment to determine if it's a go:linkname
// directive then returns the linkname information, otherwise returns nil.
func readLinknameFromComment(pkgPath string, comment *ast.Comment) (*GoLinkname, error) {
	if !strings.HasPrefix(comment.Text, `//go:linkname `) {
		return nil, nil // Not a linkname compiler directive.
	}

	fields := strings.Fields(comment.Text)

	// Check that the directive comment has both parts and is on the line by itself.
	switch len(fields) {
	case 2:
		// Ignore one-argument form //go:linkname localName
		// This is typically used with "insert"-style links to
		// suppresses the usual error for a function that lacks a body.
		// The "insert"-style links aren't supported by GopherJS so
		// these bodiless functions have to be overridden in the natives anyway.
		return nil, nil
	case 3:
		// Continue for two-argument form //go:linkname localName importPath.extName
		break
	default:
		return nil, fmt.Errorf(`gopherjs: usage requires 2 arguments: //go:linkname localName importPath.extName`)
	}

	localPkg, localName := pkgPath, fields[1]
	extPkg, extName := ``, fields[2]

	if localName == extName {
		// Ignore self referencing links, //go:linkname localName localName
		// These function similar to one-argument links.
		return nil, nil
	}

	pathOffset := 0
	if pos := strings.LastIndexByte(extName, '/'); pos != -1 {
		pathOffset = pos + 1
	}

	if idx := strings.IndexByte(extName[pathOffset:], '.'); idx != -1 {
		extPkg, extName = extName[:pathOffset+idx], extName[pathOffset+idx+1:]
	}

	return &GoLinkname{
		Reference:      symbol.Name{PkgPath: localPkg, Name: localName},
		Implementation: symbol.Name{PkgPath: extPkg, Name: extName},
	}, nil
}

// isMitigatedVarLinkname checks if the given go:linkname directive on
// a variable, which GopherJS doesn't support, is known about.
// We silently ignore such directives, since it doesn't seem to cause any problems.
func isMitigatedVarLinkname(sym symbol.Name) bool {
	mitigatedLinks := map[string]bool{
		`reflect.zeroVal`:         true,
		`math/bits.overflowError`: true, // Defaults in bits_errors_bootstrap.go
		`math/bits.divideError`:   true, // Defaults in bits_errors_bootstrap.go
		`runtime/cgo._iscgo`:      true, // Defaults in iscgo.go
	}
	return mitigatedLinks[sym.String()]
}

// isMitigatedInsertLinkname checks if the given go:linkname directive
// on a function, where the function has a body, is known about.
// These are unsupported "insert"-style go:linkname directives,
// that we ignore as a link and handle case-by-case in native overrides.
func isMitigatedInsertLinkname(sym symbol.Name) bool {
	mitigatedPkg := map[string]bool{
		`runtime`:       true, // Lots of "insert"-style links
		`internal/fuzz`: true, // Defaults to no-op stubs
	}
	mitigatedLinks := map[string]bool{
		`internal/bytealg.runtime_cmpstring`: true,
		`os.net_newUnixFile`:                 true,
	}
	return mitigatedPkg[sym.PkgPath] || mitigatedLinks[sym.String()]
}

// parseGoLinknames processed comments in a source file and extracts //go:linkname
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
func parseGoLinknames(fset *token.FileSet, pkgPath string, file *ast.File) ([]GoLinkname, error) {
	var errs ErrorList = nil
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
			return fmt.Errorf("//go:linkname local symbol %q is not found in the current source file", link.Reference.String())
		}

		if obj.Kind != ast.Fun {
			if isMitigatedVarLinkname(link.Reference) {
				return nil
			}
			return fmt.Errorf("gopherjs: //go:linkname is only supported for functions, got %q for %q", obj.Kind, link.Reference.String())
		}

		if decl := obj.Decl.(*ast.FuncDecl); decl.Body != nil {
			if isMitigatedInsertLinkname(link.Reference) {
				return nil
			}
			return fmt.Errorf("gopherjs: //go:linkname can not insert local implementation into an external package %q from local %q", link.Implementation.String(), link.Reference.String())
		}

		// Local function has no body, treat it as a reference to an external implementation.
		directives = append(directives, *link)
		return nil
	}

	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if err := processComment(c); err != nil {
				errs = append(errs, ErrorAt(err, fset, c.Pos()))
			}
		}
	}

	return directives, errs.ErrOrNil()
}

// goLinknameSet is a utility that enables quick lookup of whether a decl is
// affected by any go:linkname directive in the program.
type goLinknameSet struct {
	byImplementation map[symbol.Name][]GoLinkname
	byReference      map[symbol.Name]GoLinkname
}

// Add more GoLinkname directives into the set.
func (gls *goLinknameSet) Add(entries []GoLinkname) error {
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
func (gls *goLinknameSet) IsImplementation(sym symbol.Name) bool {
	_, found := gls.byImplementation[sym]
	return found
}

// FindImplementation returns a symbol name, which provides the implementation
// for the given symbol. The second value indicates whether the implementation
// was found.
func (gls *goLinknameSet) FindImplementation(sym symbol.Name) (symbol.Name, bool) {
	directive, found := gls.byReference[sym]
	return directive.Implementation, found
}
