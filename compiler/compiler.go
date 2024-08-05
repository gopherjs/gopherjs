// Package compiler implements GopherJS compiler logic.
//
// WARNING: This package's API is treated as internal and currently doesn't
// provide any API stability guarantee, use it at your own risk. If you need a
// stable interface, prefer invoking the gopherjs CLI tool as a subprocess.
package compiler

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"go/token"
	"go/types"
	"io"
	"strings"
	"time"

	"github.com/gopherjs/gopherjs/compiler/prelude"
	"github.com/gopherjs/gopherjs/internal/sourcemapx"
	"golang.org/x/tools/go/gcexportdata"
)

var (
	sizes32          = &types.StdSizes{WordSize: 4, MaxAlign: 8}
	reservedKeywords = make(map[string]bool)
)

func init() {
	for _, keyword := range []string{"abstract", "arguments", "boolean", "break", "byte", "case", "catch", "char", "class", "const", "continue", "debugger", "default", "delete", "do", "double", "else", "enum", "eval", "export", "extends", "false", "final", "finally", "float", "for", "function", "goto", "if", "implements", "import", "in", "instanceof", "int", "interface", "let", "long", "native", "new", "null", "package", "private", "protected", "public", "return", "short", "static", "super", "switch", "synchronized", "this", "throw", "throws", "transient", "true", "try", "typeof", "undefined", "var", "void", "volatile", "while", "with", "yield"} {
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
	// Serialized contents of go/types.Package in a binary format. This information
	// is used by the compiler to type-check packages that import this one. See
	// gcexportdata.Write().
	//
	// TODO(nevkontakte): It would be more convenient to store go/types.Package
	// itself and only serialize it when writing the archive onto disk.
	ExportData []byte
	// Compiled package-level symbols.
	Declarations []*Decl
	// Concatenated contents of all raw .inc.js of the package.
	IncJSCode []byte
	// JSON-serialized contents of go/token.FileSet. This is used to obtain source
	// code locations for various symbols (e.g. for sourcemap generation). See
	// token.FileSet.Write().
	//
	// TODO(nevkontakte): This is also more convenient to store as the original
	// object and only serialize before writing onto disk.
	FileSet []byte
	// Whether or not the package was compiled with minification enabled.
	Minified bool
	// A list of go:linkname directives encountered in the package.
	GoLinknames []GoLinkname
	// Time when this archive was built.
	BuildTime time.Time
}

func (a Archive) String() string {
	return fmt.Sprintf("compiler.Archive{%s}", a.ImportPath)
}

// RegisterTypes adds package type information from the archive into the provided map.
func (a *Archive) RegisterTypes(packages map[string]*types.Package) error {
	var err error
	// TODO(nevkontakte): Should this be shared throughout the build?
	fset := token.NewFileSet()
	packages[a.ImportPath], err = gcexportdata.Read(bytes.NewReader(a.ExportData), fset, packages, a.ImportPath)
	return err
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

type dceInfo struct {
	decl         *Decl
	objectFilter string
	methodFilter string
}

func WriteProgramCode(pkgs []*Archive, w *sourcemapx.Filter, goVersion string) error {
	mainPkg := pkgs[len(pkgs)-1]
	minify := mainPkg.Minified

	// Aggregate all go:linkname directives in the program together.
	gls := goLinknameSet{}
	for _, pkg := range pkgs {
		gls.Add(pkg.GoLinknames)
	}

	byFilter := make(map[string][]*dceInfo)
	var pendingDecls []*Decl // A queue of live decls to find other live decls.
	for _, pkg := range pkgs {
		for _, d := range pkg.Declarations {
			if d.DceObjectFilter == "" && d.DceMethodFilter == "" {
				// This is an entry point (like main() or init() functions) or a variable
				// initializer which has a side effect, consider it live.
				pendingDecls = append(pendingDecls, d)
				continue
			}
			if gls.IsImplementation(d.LinkingName) {
				// If a decl is referenced by a go:linkname directive, we just assume
				// it's not dead.
				// TODO(nevkontakte): This is a safe, but imprecise assumption. We should
				// try and trace whether the referencing functions are actually live.
				pendingDecls = append(pendingDecls, d)
			}
			info := &dceInfo{decl: d}
			if d.DceObjectFilter != "" {
				info.objectFilter = pkg.ImportPath + "." + d.DceObjectFilter
				byFilter[info.objectFilter] = append(byFilter[info.objectFilter], info)
			}
			if d.DceMethodFilter != "" {
				info.methodFilter = pkg.ImportPath + "." + d.DceMethodFilter
				byFilter[info.methodFilter] = append(byFilter[info.methodFilter], info)
			}
		}
	}

	dceSelection := make(map[*Decl]struct{}) // Known live decls.
	for len(pendingDecls) != 0 {
		d := pendingDecls[len(pendingDecls)-1]
		pendingDecls = pendingDecls[:len(pendingDecls)-1]

		dceSelection[d] = struct{}{} // Mark the decl as live.

		// Consider all decls the current one is known to depend on and possible add
		// them to the live queue.
		for _, dep := range d.DceDeps {
			if infos, ok := byFilter[dep]; ok {
				delete(byFilter, dep)
				for _, info := range infos {
					if info.objectFilter == dep {
						info.objectFilter = ""
					}
					if info.methodFilter == dep {
						info.methodFilter = ""
					}
					if info.objectFilter == "" && info.methodFilter == "" {
						pendingDecls = append(pendingDecls, info.decl)
					}
				}
			}
		}
	}

	if _, err := w.Write([]byte("\"use strict\";\n(function() {\n\n")); err != nil {
		return err
	}
	if _, err := w.Write([]byte(fmt.Sprintf("var $goVersion = %q;\n", goVersion))); err != nil {
		return err
	}

	preludeJS := prelude.Prelude
	if minify {
		preludeJS = prelude.Minified()
	}
	if _, err := io.WriteString(w, preludeJS); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}

	// write packages
	for _, pkg := range pkgs {
		if err := WritePkgCode(pkg, dceSelection, gls, minify, w); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte("$synthesizeMethods();\n$initAllLinknames();\nvar $mainPkg = $packages[\"" + string(mainPkg.ImportPath) + "\"];\n$packages[\"runtime\"].$init();\n$go($mainPkg.$init, []);\n$flushConsole();\n\n}).call(this);\n")); err != nil {
		return err
	}
	return nil
}

func WritePkgCode(pkg *Archive, dceSelection map[*Decl]struct{}, gls goLinknameSet, minify bool, w *sourcemapx.Filter) error {
	if w.MappingCallback != nil && pkg.FileSet != nil {
		w.FileSet = token.NewFileSet()
		if err := w.FileSet.Read(json.NewDecoder(bytes.NewReader(pkg.FileSet)).Decode); err != nil {
			panic(err)
		}
	}
	if _, err := w.Write(pkg.IncJSCode); err != nil {
		return err
	}
	if _, err := w.Write(removeWhitespace([]byte(fmt.Sprintf("$packages[\"%s\"] = (function() {\n", pkg.ImportPath)), minify)); err != nil {
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
	if _, err := w.Write(removeWhitespace([]byte(fmt.Sprintf("\tvar %s;\n", strings.Join(vars, ", "))), minify)); err != nil {
		return err
	}
	for _, d := range filteredDecls {
		if _, err := w.Write(d.DeclCode); err != nil {
			return err
		}
		if gls.IsImplementation(d.LinkingName) {
			// This decl is referenced by a go:linkname directive, expose it to external
			// callers via $linkname object (declared in prelude). We are not using
			// $pkg to avoid clashes with exported symbols.
			var code string
			if recv, method, ok := d.LinkingName.IsMethod(); ok {
				code = fmt.Sprintf("\t$linknames[%q] = $unsafeMethodToFunction(%v,%q,%t);\n", d.LinkingName.String(), d.NamedRecvType, method, strings.HasPrefix(recv, "*"))
			} else {
				code = fmt.Sprintf("\t$linknames[%q] = %s;\n", d.LinkingName.String(), d.RefExpr)
			}
			if _, err := w.Write(removeWhitespace([]byte(code), minify)); err != nil {
				return err
			}
		}
	}
	for _, d := range filteredDecls {
		if _, err := w.Write(d.MethodListCode); err != nil {
			return err
		}
	}
	for _, d := range filteredDecls {
		if _, err := w.Write(d.TypeInitCode); err != nil {
			return err
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
			lines = append(lines, fmt.Sprintf("\t\t%s = $linknames[%q];\n", d.RefExpr, impl.String()))
		}
		if len(lines) > 0 {
			code := fmt.Sprintf("\t$pkg.$initLinknames = function() {\n%s};\n", strings.Join(lines, ""))
			if _, err := w.Write(removeWhitespace([]byte(code), minify)); err != nil {
				return err
			}
		}
	}

	if _, err := w.Write(removeWhitespace([]byte("\t$init = function() {\n\t\t$pkg.$init = function() {};\n\t\t/* */ var $f, $c = false, $s = 0, $r; if (this !== undefined && this.$blk !== undefined) { $f = this; $c = true; $s = $f.$s; $r = $f.$r; } s: while (true) { switch ($s) { case 0:\n"), minify)); err != nil {
		return err
	}
	for _, d := range filteredDecls {
		if _, err := w.Write(d.InitCode); err != nil {
			return err
		}
	}
	if _, err := w.Write(removeWhitespace([]byte("\t\t/* */ } return; } if ($f === undefined) { $f = { $blk: $init }; } $f.$s = $s; $f.$r = $r; return $f;\n\t};\n\t$pkg.$init = $init;\n\treturn $pkg;\n})();"), minify)); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil { // keep this \n even when minified
		return err
	}
	return nil
}

// ReadArchive reads serialized compiled archive of the importPath package.
func ReadArchive(path string, r io.Reader) (*Archive, error) {
	var a Archive
	if err := gob.NewDecoder(r).Decode(&a); err != nil {
		return nil, err
	}

	return &a, nil
}

// WriteArchive writes compiled package archive on disk for later reuse.
func WriteArchive(a *Archive, w io.Writer) error {
	return gob.NewEncoder(w).Encode(a)
}
