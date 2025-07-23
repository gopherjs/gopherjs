// Package compiler implements GopherJS compiler logic.
//
// WARNING: This package's API is treated as internal and currently doesn't
// provide any API stability guarantee, use it at your own risk. If you need a
// stable interface, prefer invoking the gopherjs CLI tool as a subprocess.
package compiler

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"go/token"
	"go/types"
	"io"
	"sort"
	"strings"
	"time"

	"golang.org/x/tools/go/gcexportdata"

	"github.com/gopherjs/gopherjs/compiler/internal/dce"
	"github.com/gopherjs/gopherjs/compiler/internal/grouper"
	"github.com/gopherjs/gopherjs/compiler/linkname"
	"github.com/gopherjs/gopherjs/compiler/prelude"
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

	// Perform dead code elimination (DCE) on the declarations
	// to get the selection of the declarations that are actually used.
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

	// TODO(grantnelson-wf): REMOVE the following debug prints
	//dot := grouper.ToGraph(dceSelection, sequencer.GraphOptions[*Decl]{
	//	ItemToString: func(d *Decl) string { return d.FullName },
	//	FilterCycles: true,
	//	StrictFilter: true,
	//})
	//if err := os.WriteFile("./dot.gv", []byte(dot), 0644); err != nil {
	//	panic(fmt.Errorf(`failed to write dot.gv: %w`, err))
	//}
	if cs := grouper.CyclesToString(dceSelection, func(d *Decl) string { return d.FullName }); cs != "" {
		fmt.Printf("Cycles detected in the dependency graph:\n%s\n", cs)
	}

	// Set the Decl.Grouper().Group values for each declaration.
	// The group number is used to determine the type initialization order.
	groupCount := grouper.Group(dceSelection)

	// TODO(grantnelson-wf): REMOVE the following debug prints
	//fmt.Println(`-------------------------->>`)
	//for _, pkg := range pkgs {
	//	fmt.Println(`--[`, pkg.Name, `]--`)
	//	for _, d := range pkg.Declarations {
	//		if _, ok := dceSelection[d]; ok {
	//			fmt.Println("\t", d.FullName, `=>`, d.Grouper().String())
	//		}
	//	}
	//}
	//fmt.Println(`<<--------------------------`)

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
		if err := WritePkgCode(pkg, dceSelection, gls, groupCount, minify, w); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(w, "$initializeTypes(%d);\n$synthesizeMethods();\n$initAllLinknames();\nvar $mainPkg = $packages[\"%s\"];\n$packages[\"runtime\"].$init();\n$go($mainPkg.$init, []);\n$flushConsole();\n\n}).call(this);\n", groupCount+1, string(mainPkg.ImportPath)); err != nil {
		return err
	}
	return nil
}

func WritePkgCode(pkg *Archive, dceSelection map[*Decl]struct{}, gls linkname.GoLinknameSet, groupCount int, minify bool, w *SourceMapFilter) error {
	if w.MappingCallback != nil && pkg.FileSet != nil {
		w.fileSet = pkg.FileSet
	}
	if _, err := w.Write(pkg.IncJSCode); err != nil {
		return err
	}
	if _, err := w.Write(removeWhitespace([]byte(fmt.Sprintf("$packages[\"%s\"] = (function() {\n", pkg.ImportPath)), minify)); err != nil {
		return err
	}

	vars := []string{"$pkg = {}", "$init"}
	var filteredDecls []*Decl
	groupMap := make(map[int][]*Decl)
	for _, d := range pkg.Declarations {
		if _, ok := dceSelection[d]; ok {
			vars = append(vars, d.Vars...)
			filteredDecls = append(filteredDecls, d)
			group := d.Grouper().Group
			groupMap[group] = append(groupMap[group], d)
		}
	}
	if _, err := w.Write(removeWhitespace([]byte(fmt.Sprintf("\tvar %s;\n", strings.Join(vars, ", "))), minify)); err != nil {
		return err
	}

	groups := make([]int, 0, len(groupMap))
	for group := range groupMap {
		groups = append(groups, group)
	}
	sort.Ints(groups)

	var firstPrint bool
	for _, group := range groups {
		groupDecls := groupMap[group]
		if _, err := w.Write(removeWhitespace([]byte(fmt.Sprintf("\t$addTypeInit(%d, this, function() {\n", group)), minify)); err != nil {
			return err
		}
		firstPrint = true
		for _, d := range groupDecls {
			if len(d.DeclCode) > 0 {
				if firstPrint { // TODO(grantnelson-wf): CLEAN UP
					firstPrint = false
					if _, err := w.Write([]byte("\n\t// ---[ Decl Code ]---\n")); err != nil {
						return err
					}
				}
				if _, err := fmt.Fprintf(w, "\t// %s\n", d.FullName); err != nil {
					return err
				}
				if _, err := w.Write(d.DeclCode); err != nil {
					return err
				}
			}
		}

		firstPrint = true
		for _, d := range groupDecls {
			if len(d.TypeInitCode) > 0 {
				if firstPrint { // TODO(grantnelson-wf): CLEAN UP
					firstPrint = false
					if _, err := w.Write([]byte("\n\t// ---[ Type Init Code ]---\n")); err != nil {
						return err
					}
				}
				if _, err := fmt.Fprintf(w, "\t// %s\n", d.FullName); err != nil {
					return err
				}
				if _, err := w.Write(d.TypeInitCode); err != nil {
					return err
				}
			}
		}

		if _, err := w.Write([]byte("\t});\n")); err != nil {
			return err
		}
	}

	// TODO(grantnelson-wf): Don't write unless there is stuff in here to write
	if _, err := w.Write(removeWhitespace([]byte(fmt.Sprintf("\t$addTypeInit(%d, this, function() { // Finalize\n", groupCount)), minify)); err != nil {
		return err
	}
	for _, d := range filteredDecls {
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

	firstPrint = true
	for _, d := range filteredDecls {
		if len(d.MethodListCode) > 0 {
			if firstPrint { // TODO(grantnelson-wf): CLEAN UP
				firstPrint = false
				if _, err := w.Write([]byte("\n\t// ---[ Method List Code ]---\n")); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(w, "\t// %s\n", d.FullName); err != nil {
				return err
			}
			if _, err := w.Write(d.MethodListCode); err != nil {
				return err
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
			lines = append(lines, fmt.Sprintf("\t\t%s = $linknames[%q];\n", d.RefExpr, impl.String()))
		}
		if len(lines) > 0 {
			code := fmt.Sprintf("\t$pkg.$initLinknames = function() {\n%s};\n", strings.Join(lines, ""))
			if _, err := w.Write(removeWhitespace([]byte(code), minify)); err != nil {
				return err
			}
		}
	}

	if _, err := w.Write([]byte("\t});\n")); err != nil {
		return err
	}

	if _, err := w.Write([]byte("\n\t// ---[ Init ]---\n")); err != nil { // TODO(grantnelson-wf): CLEAN UP
		return err
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

type serializableArchive struct {
	ImportPath   string
	Name         string
	Imports      []string
	ExportData   []byte
	Declarations []*Decl
	IncJSCode    []byte
	FileSet      []byte
	Minified     bool
	GoLinknames  []linkname.GoLinkname
	BuildTime    time.Time
}

// ReadArchive reads serialized compiled archive of the importPath package.
//
// The given srcModTime is used to determine if the archive is out-of-date.
// If the archive is out-of-date, the returned archive is nil.
// If there was not an error, the returned time is when the archive was built.
//
// The imports map is used to resolve package dependencies and may modify the
// map to include the package from the read archive. See [gcexportdata.Read].
func ReadArchive(importPath string, r io.Reader, srcModTime time.Time, imports map[string]*types.Package) (*Archive, time.Time, error) {
	var sa serializableArchive
	if err := gob.NewDecoder(r).Decode(&sa); err != nil {
		return nil, time.Time{}, err
	}

	if srcModTime.After(sa.BuildTime) {
		// Archive is out-of-date.
		return nil, sa.BuildTime, nil
	}

	var a Archive
	fset := token.NewFileSet()
	if len(sa.ExportData) > 0 {
		pkg, err := gcexportdata.Read(bytes.NewReader(sa.ExportData), fset, imports, importPath)
		if err != nil {
			return nil, sa.BuildTime, err
		}
		a.Package = pkg
	}

	if len(sa.FileSet) > 0 {
		a.FileSet = token.NewFileSet()
		if err := a.FileSet.Read(json.NewDecoder(bytes.NewReader(sa.FileSet)).Decode); err != nil {
			return nil, sa.BuildTime, err
		}
	}

	a.ImportPath = sa.ImportPath
	a.Name = sa.Name
	a.Imports = sa.Imports
	a.Declarations = sa.Declarations
	a.IncJSCode = sa.IncJSCode
	a.Minified = sa.Minified
	a.GoLinknames = sa.GoLinknames
	return &a, sa.BuildTime, nil
}

// WriteArchive writes compiled package archive on disk for later reuse.
//
// The passed in buildTime is used to determine if the archive is out-of-date.
// Typically it should be set to the srcModTime or time.Now() but it is exposed for testing purposes.
func WriteArchive(a *Archive, buildTime time.Time, w io.Writer) error {
	exportData := new(bytes.Buffer)
	if a.Package != nil {
		if err := gcexportdata.Write(exportData, nil, a.Package); err != nil {
			return fmt.Errorf("failed to write export data: %w", err)
		}
	}

	encodedFileSet := new(bytes.Buffer)
	if a.FileSet != nil {
		if err := a.FileSet.Write(json.NewEncoder(encodedFileSet).Encode); err != nil {
			return err
		}
	}

	sa := serializableArchive{
		ImportPath:   a.ImportPath,
		Name:         a.Name,
		Imports:      a.Imports,
		ExportData:   exportData.Bytes(),
		Declarations: a.Declarations,
		IncJSCode:    a.IncJSCode,
		FileSet:      encodedFileSet.Bytes(),
		Minified:     a.Minified,
		GoLinknames:  a.GoLinknames,
		BuildTime:    buildTime,
	}

	return gob.NewEncoder(w).Encode(sa)
}

type SourceMapFilter struct {
	Writer          io.Writer
	MappingCallback func(generatedLine, generatedColumn int, originalPos token.Position)
	line            int
	column          int
	fileSet         *token.FileSet
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
