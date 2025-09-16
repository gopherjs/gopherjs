// Package compiler implements GopherJS compiler logic.
//
// WARNING: This package's API is treated as internal and currently doesn't
// provide any API stability guarantee, use it at your own risk. If you need a
// stable interface, prefer invoking the gopherjs CLI tool as a subprocess.
package compiler

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go/token"
	"go/types"
	"io"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/internal/dce"
	"github.com/gopherjs/gopherjs/compiler/linkname"
	"github.com/gopherjs/gopherjs/compiler/prelude"
)

var (
	sizes32          = &types.StdSizes{WordSize: 4, MaxAlign: 8}
	reservedKeywords = make(map[string]bool)
)

func init() {
	keywords := []string{
		"abstract", "arguments", "boolean", "break", "byte", "case", "catch", "char", "class", "const", "continue",
		"debugger", "default", "delete", "do", "double", "else", "enum", "eval", "export", "extends", "false",
		"final", "finally", "float", "for", "function", "goto", "if", "implements", "import", "in", "instanceof",
		"int", "interface", "let", "long", "native", "new", "null", "package", "private", "protected", "public",
		"return", "short", "static", "super", "switch", "synchronized", "this", "throw", "throws", "transient",
		"true", "try", "typeof", "undefined", "var", "void", "volatile", "while", "with", "yield",
	}
	for _, keyword := range keywords {
		reservedKeywords[keyword] = true
	}
}

// Archive contains intermediate build outputs of a single package.
//
// This is a logical equivalent of an object file in traditional compilers.
type Archive struct {
	// Package's full import path, e.g. "some/package/name".
	ImportPath string
	// Package's name as per "package" statement at the top of a source file.
	// Usually matches the last component of import path, but may differ in
	// certain cases (e.g. main or test packages).
	Name string
	// A list of full package import paths that the current package imports across
	// all source files. See go/types.Package.Imports().
	Imports []string
	// The package information is used by the compiler to type-check packages
	// that import this one. See [gcexportdata.Write].
	Package *types.Package
	// Compiled package-level symbols.
	Declarations []*Decl
	// Concatenated contents of all raw .inc.js of the package.
	IncJSCode []byte
	// The file set containing the source code locations for various symbols
	// (e.g. for sourcemap generation). See [token.FileSet.Write].
	FileSet *token.FileSet
	// Whether or not the package was compiled with minification enabled.
	Minified bool
	// A list of go:linkname directives encountered in the package.
	GoLinknames []linkname.GoLinkname
}

func (a Archive) String() string {
	return fmt.Sprintf("compiler.Archive{%s}", a.ImportPath)
}

type Dependency struct {
	Pkg    string
	Type   string
	Method string
}

func ImportDependencies(archive *Archive, importPkg func(string) (*Archive, error)) ([]*Archive, error) {
	var deps []*Archive
	paths := make(map[string]bool)
	var collectDependencies func(path string) error
	collectDependencies = func(path string) error {
		if paths[path] {
			return nil
		}
		dep, err := importPkg(path)
		if err != nil {
			return err
		}
		for _, imp := range dep.Imports {
			if err := collectDependencies(imp); err != nil {
				return err
			}
		}
		deps = append(deps, dep)
		paths[dep.ImportPath] = true
		return nil
	}

	if err := collectDependencies("runtime"); err != nil {
		return nil, err
	}
	for _, imp := range archive.Imports {
		if err := collectDependencies(imp); err != nil {
			return nil, err
		}
	}

	deps = append(deps, archive)
	return deps, nil
}

func WriteProgramCode(pkgs []*Archive, w *SourceMapFilter, goVersion string) error {
	mainPkg := pkgs[len(pkgs)-1]
	minify := mainPkg.Minified

	// Aggregate all go:linkname directives in the program together.
	gls := linkname.GoLinknameSet{}
	for _, pkg := range pkgs {
		gls.Add(pkg.GoLinknames)
	}

	sel := &dce.Selector[*Decl]{}
	for _, pkg := range pkgs {
		for _, d := range pkg.Declarations {
			implementsLink := false
			if gls.IsImplementation(d.LinkingName) {
				// If a decl is referenced by a go:linkname directive, we just assume
				// it's not dead.
				// TODO(nevkontakte): This is a safe, but imprecise assumption. We should
				// try and trace whether the referencing functions are actually live.
				implementsLink = true
			}
			sel.Include(d, implementsLink)
		}
	}
	dceSelection := sel.AliveDecls()

	if _, err := w.WriteF(false, "\"use strict\";\n(function() {\n\n"); err != nil {
		return err
	}
	if _, err := w.WriteF(false, "var $goVersion = %q;\n", goVersion); err != nil {
		return err
	}

	preludeJS := prelude.Prelude
	if minify {
		preludeJS = prelude.Minified()
	}
	if _, err := io.WriteString(w, preludeJS); err != nil {
		return err
	}
	if _, err := w.WriteF(false, "\n"); err != nil {
		return err
	}

	// write packages
	for _, pkg := range pkgs {
		if err := WritePkgCode(pkg, dceSelection, gls, minify, w); err != nil {
			return err
		}
	}

	if _, err := w.WriteF(false, "$callForAllPackages(\"$finishSetup\");\n"); err != nil {
		return err
	}
	if _, err := w.WriteF(false, "$synthesizeMethods();\n"); err != nil {
		return err
	}
	if _, err := w.WriteF(false, "$callForAllPackages(\"$initLinknames\");\n"); err != nil {
		return err
	}
	if _, err := w.WriteF(false, "var $mainPkg = $packages[\"%s\"];\n", mainPkg.ImportPath); err != nil {
		return err
	}
	if _, err := w.WriteF(false, "$packages[\"runtime\"].$init();\n"); err != nil {
		return err
	}
	if _, err := w.WriteF(false, "$go($mainPkg.$init, []);\n"); err != nil {
		return err
	}
	if _, err := w.WriteF(false, "$flushConsole();\n"); err != nil {
		return err
	}
	if _, err := w.WriteF(false, "\n}).call(this);\n"); err != nil {
		return err
	}
	return nil
}

func WritePkgCode(pkg *Archive, dceSelection map[*Decl]struct{}, gls linkname.GoLinknameSet, minify bool, w *SourceMapFilter) error {
	if w.MappingCallback != nil && pkg.FileSet != nil {
		w.fileSet = pkg.FileSet
	}
	if _, err := w.Write(pkg.IncJSCode); err != nil {
		return err
	}
	if _, err := w.WriteF(minify, "$packages[\"%s\"] = (function() {\n", pkg.ImportPath); err != nil {
		return err
	}
	vars := []string{"$pkg = {}", "$init"}
	var filteredDecls []*Decl
	for _, d := range pkg.Declarations {
		if _, ok := dceSelection[d]; ok {
			vars = append(vars, d.Vars...)
			filteredDecls = append(filteredDecls, d)
		}
	}
	// Write variable names
	if _, err := w.WriteF(minify, "\tvar %s;\n", strings.Join(vars, ", ")); err != nil {
		return err
	}
	// Write imports
	for _, d := range filteredDecls {
		if _, err := w.Write(d.ImportCode); err != nil {
			return err
		}
	}
	// Write named type declarations
	for _, d := range filteredDecls {
		if _, err := w.Write(d.TypeDeclCode); err != nil {
			return err
		}
	}
	// Write exports for named type declarations
	for _, d := range filteredDecls {
		if _, err := w.Write(d.ExportTypeCode); err != nil {
			return err
		}
	}

	// The following parts have to be run after all packages have been added
	// to handle generics that use named types defined in a package that
	// is defined after this package has been defined.
	if _, err := w.WriteF(minify, "\t$pkg.$finishSetup = function() {\n"); err != nil {
		return err
	}

	// Write anonymous type declarations
	for _, d := range filteredDecls {
		if _, err := w.Write(d.AnonTypeDeclCode); err != nil {
			return err
		}
	}
	// Write function declarations
	for _, d := range filteredDecls {
		if _, err := w.Write(d.FuncDeclCode); err != nil {
			return err
		}
	}
	// Write exports for function declarations
	for _, d := range filteredDecls {
		if _, err := w.Write(d.ExportFuncCode); err != nil {
			return err
		}
	}
	// Write reflection metadata for types' methods
	for _, d := range filteredDecls {
		if _, err := w.Write(d.MethodListCode); err != nil {
			return err
		}
	}
	// Write the calls to finish initialization of types
	for _, d := range filteredDecls {
		if _, err := w.Write(d.TypeInitCode); err != nil {
			return err
		}
	}

	for _, d := range filteredDecls {
		if gls.IsImplementation(d.LinkingName) {
			// This decl is referenced by a go:linkname directive, expose it to external
			// callers via $linkname object (declared in prelude). We are not using
			// $pkg to avoid clashes with exported symbols.
			if recv, method, ok := d.LinkingName.IsMethod(); ok {
				if _, err := w.WriteF(minify, "\t$linknames[%q] = $unsafeMethodToFunction(%v,%q,%t);\n", d.LinkingName.String(), d.NamedRecvType, method, strings.HasPrefix(recv, "*")); err != nil {
					return err
				}
			} else {
				if _, err := w.WriteF(minify, "\t$linknames[%q] = %s;\n", d.LinkingName.String(), d.RefExpr); err != nil {
					return err
				}
			}
		}
	}

	{
		// Set up all functions which package declares, but which implementation
		// comes from elsewhere via a go:linkname compiler directive. This code
		// needs to be executed after all $packages entries were defined, since such
		// reference may go in a direction opposite of the import graph. It also
		// needs to run before any initializer code runs, since that code may invoke
		// linknamed function.
		lines := []string{}
		for _, d := range filteredDecls {
			impl, found := gls.FindImplementation(d.LinkingName)
			if !found {
				continue // The symbol is not affected by a go:linkname directive.
			}
			lines = append(lines, fmt.Sprintf("\t\t\t%s = $linknames[%q];\n", d.RefExpr, impl.String()))
		}
		if len(lines) > 0 {
			if _, err := w.WriteF(minify, "\t\t$pkg.$initLinknames = function() {\n%s};\n", strings.Join(lines, "")); err != nil {
				return err
			}
		}
	}

	// Write the end of the `$finishSetup` function.
	if _, err := w.WriteF(minify, "\t};\n"); err != nil {
		return err
	}

	// Write the initialization function that will initialize this package
	// (e.g. initialize package-level variable value).
	if _, err := w.WriteF(minify, "\t$init = function() {\n"); err != nil {
		return err
	}
	if _, err := w.WriteF(minify, "\t\t$pkg.$init = function() {};\n"); err != nil {
		return err
	}
	if _, err := w.WriteF(minify, "\t\t/* */ var $f, $c = false, $s = 0, $r; if (this !== undefined && this.$blk !== undefined) { $f = this; $c = true; $s = $f.$s; $r = $f.$r; } s: while (true) { switch ($s) { case 0:\n"); err != nil {
		return err
	}
	for _, d := range filteredDecls {
		if _, err := w.Write(d.InitCode); err != nil {
			return err
		}
	}
	if _, err := w.WriteF(minify, "\t\t/* */ } return; } if ($f === undefined) { $f = { $blk: $init }; } $f.$s = $s; $f.$r = $r; return $f;\n"); err != nil {
		return err
	}
	if _, err := w.WriteF(minify, "\t};\n"); err != nil {
		return err
	}
	if _, err := w.WriteF(minify, "\t$pkg.$init = $init;\n"); err != nil {
		return err
	}
	if _, err := w.WriteF(minify, "\treturn $pkg;\n"); err != nil {
		return err
	}
	if _, err := w.WriteF(minify, "})();"); err != nil {
		return err
	}
	if _, err := w.WriteF(false, "\n"); err != nil { // keep this \n even when minified
		return err
	}
	return nil
}

type SourceMapFilter struct {
	Writer          io.Writer
	MappingCallback func(generatedLine, generatedColumn int, originalPos token.Position)
	line            int
	column          int
	fileSet         *token.FileSet
}

func (f *SourceMapFilter) WriteF(minify bool, format string, args ...any) (n int, err error) {
	return f.Write(removeWhitespace([]byte(fmt.Sprintf(format, args...)), minify))
}

func (f *SourceMapFilter) Write(p []byte) (n int, err error) {
	var n2 int
	for {
		i := bytes.IndexByte(p, '\b')
		w := p
		if i != -1 {
			w = p[:i]
		}

		n2, err = f.Writer.Write(w)
		n += n2
		for {
			i := bytes.IndexByte(w, '\n')
			if i == -1 {
				f.column += len(w)
				break
			}
			f.line++
			f.column = 0
			w = w[i+1:]
		}

		if err != nil || i == -1 {
			return
		}
		if f.MappingCallback != nil {
			f.MappingCallback(f.line+1, f.column, f.fileSet.Position(token.Pos(binary.BigEndian.Uint32(p[i+1:i+5]))))
		}
		p = p[i+5:]
		n += 5
	}
}
