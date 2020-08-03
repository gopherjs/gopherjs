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
	"strings"

	"github.com/gopherjs/gopherjs/compiler/prelude"
	"golang.org/x/tools/go/gcexportdata"
)

var sizes32 = &types.StdSizes{WordSize: 4, MaxAlign: 8}
var reservedKeywords = make(map[string]bool)

func init() {
	for _, keyword := range []string{"abstract", "arguments", "boolean", "break", "byte", "case", "catch", "char", "class", "const", "continue", "debugger", "default", "delete", "do", "double", "else", "enum", "eval", "export", "extends", "false", "final", "finally", "float", "for", "function", "goto", "if", "implements", "import", "in", "instanceof", "int", "interface", "let", "long", "native", "new", "null", "package", "private", "protected", "public", "return", "short", "static", "super", "switch", "synchronized", "this", "throw", "throws", "transient", "true", "try", "typeof", "undefined", "var", "void", "volatile", "while", "with", "yield"} {
		reservedKeywords[keyword] = true
	}
}

type ErrorList []error

func (err ErrorList) Error() string {
	return err[0].Error()
}

type Archive struct {
	ImportPath   string
	Name         string
	Imports      []string
	ExportData   []byte
	Declarations []*Decl
	IncJSCode    []byte
	FileSet      []byte
	Minified     bool
}

type Decl struct {
	FullName        string
	Vars            []string
	DeclCode        []byte
	MethodListCode  []byte
	TypeInitCode    []byte
	InitCode        []byte
	DceObjectFilter string
	DceMethodFilter string
	DceDeps         []string
	Blocking        bool
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

func WriteProgramCode(pkgs []*Archive, w *SourceMapFilter) error {
	mainPkg := pkgs[len(pkgs)-1]
	minify := mainPkg.Minified

	byFilter := make(map[string][]*dceInfo)
	var pendingDecls []*Decl
	for _, pkg := range pkgs {
		for _, d := range pkg.Declarations {
			if d.DceObjectFilter == "" && d.DceMethodFilter == "" {
				pendingDecls = append(pendingDecls, d)
				continue
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

	dceSelection := make(map[*Decl]struct{})
	for len(pendingDecls) != 0 {
		d := pendingDecls[len(pendingDecls)-1]
		pendingDecls = pendingDecls[:len(pendingDecls)-1]

		dceSelection[d] = struct{}{}

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

	leaveSection := w.EnterSection("$prelude")
	defer leaveSection()

	if _, err := w.Write([]byte("\"use strict\";\n(function() {\n\n")); err != nil {
		return err
	}
	preludeJS := prelude.Prelude
	if minify {
		preludeJS = prelude.Minified
	}
	if _, err := io.WriteString(w, preludeJS); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}

	// write packages
	for _, pkg := range pkgs {
		if err := WritePkgCode(pkg, dceSelection, minify, w); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte("$synthesizeMethods();\nvar $mainPkg = $packages[\"" + string(mainPkg.ImportPath) + "\"];\n$packages[\"runtime\"].$init();\n$go($mainPkg.$init, []);\n$flushConsole();\n\n}).call(this);\n")); err != nil {
		return err
	}

	return nil
}

func WritePkgCode(pkg *Archive, dceSelection map[*Decl]struct{}, minify bool, w *SourceMapFilter) error {
	leaveSection := w.EnterSection(pkg.ImportPath)
	defer leaveSection()

	if w.MappingCallback != nil && pkg.FileSet != nil {
		w.fileSet = token.NewFileSet()
		if err := w.fileSet.Read(json.NewDecoder(bytes.NewReader(pkg.FileSet)).Decode); err != nil {
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

func ReadArchive(filename, path string, r io.Reader, packages map[string]*types.Package) (*Archive, error) {
	var a Archive
	if err := gob.NewDecoder(r).Decode(&a); err != nil {
		return nil, err
	}

	var err error
	packages[path], err = gcexportdata.Read(bytes.NewReader(a.ExportData), token.NewFileSet(), packages, path)
	if err != nil {
		return nil, err
	}

	return &a, nil
}

func WriteArchive(a *Archive, w io.Writer) error {
	return gob.NewEncoder(w).Encode(a)
}

type SourceMapFilter struct {
	Writer          io.Writer
	MappingCallback func(generatedLine, generatedColumn int, originalPos token.Position)
	line            int
	column          int
	fileSet         *token.FileSet
	// BytesWritten per program section.
	//
	// This information is used to produce artifact size report. The key is
	// a Go package name or one of a special IDs: $prelude.
	BytesWritten map[string]int
	// The current program section being written. Empty string means "UNKNOWN".
	currentSection string
}

// Write program code and populate source map if available.
//
// The passed byte slice is written verbatim, except when it contains an ASCII
// "backspace" character (\b, 0x08). In that case the 4 bytes following the
// backspace character are interpreted as big-endian-encoded token.Pos that must
// be present in the fileSet. The position is interpreted as the position in the
// original source code for the JS code that is about to be written (?) and will
// be passed to the MappingCallback in order to emit the appropriate source map.
// The sequence of \b and the following 4 bytes are then omitted from the actual
// output.
func (f *SourceMapFilter) Write(p []byte) (n int, err error) {
	if f.BytesWritten == nil {
		f.BytesWritten = map[string]int{}
	}

	var n2 int
	for {
		i := bytes.IndexByte(p, '\b')
		w := p
		if i != -1 {
			w = p[:i]
		}

		n2, err = f.Writer.Write(w)
		n += n2
		f.BytesWritten[f.currentSection] += n2
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

// EnterSection marks a beginning of a new program section.
//
// All bytes written between entering and leaving a section will be attributed
// to the specified section in the artifact size report. The returned callback
// restores the previous section, allowing for sections to nest.
func (f *SourceMapFilter) EnterSection(s string) func() {
	previousSection := f.currentSection
	f.currentSection = s
	return func() { f.currentSection = previousSection }
}
