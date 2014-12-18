package compiler

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"go/token"
	"io"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/prelude"
	"golang.org/x/tools/go/gcimporter"
	"golang.org/x/tools/go/types"
)

var sizes32 = &types.StdSizes{WordSize: 4, MaxAlign: 8}
var reservedKeywords = make(map[string]bool)

func init() {
	for _, keyword := range []string{"abstract", "arguments", "boolean", "break", "byte", "case", "catch", "char", "class", "const", "continue", "debugger", "default", "delete", "do", "double", "else", "enum", "eval", "export", "extends", "false", "final", "finally", "float", "for", "function", "goto", "if", "implements", "import", "in", "instanceof", "int", "interface", "let", "long", "native", "new", "package", "private", "protected", "public", "return", "short", "static", "super", "switch", "synchronized", "this", "throw", "throws", "transient", "true", "try", "typeof", "var", "void", "volatile", "while", "with", "yield"} {
		reservedKeywords[keyword] = true
	}
}

type ErrorList []error

func (err ErrorList) Error() string {
	return err[0].Error()
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
			if err := collectDependencies(imp.Path); err != nil {
				return err
			}
		}
		deps = append(deps, dep)
		paths[dep.ImportPath] = true
		return nil
	}

	collectDependencies("runtime")
	for _, imp := range archive.Imports {
		if err := collectDependencies(imp.Path); err != nil {
			return nil, err
		}
	}

	deps = append(deps, archive)
	return deps, nil
}

func WriteProgramCode(pkgs []*Archive, w *SourceMapFilter) error {
	mainPkg := pkgs[len(pkgs)-1]
	minify := mainPkg.Minified

	declsByObject := make(map[string][]*Decl)
	var pendingDecls []*Decl
	for _, pkg := range pkgs {
		for _, d := range pkg.Declarations {
			if len(d.DceFilters) == 0 {
				pendingDecls = append(pendingDecls, d)
				continue
			}
			for _, f := range d.DceFilters {
				o := string(pkg.ImportPath) + ":" + string(f)
				declsByObject[o] = append(declsByObject[o], d)
			}
		}
	}

	for len(pendingDecls) != 0 {
		d := pendingDecls[len(pendingDecls)-1]
		pendingDecls = pendingDecls[:len(pendingDecls)-1]
		for _, dep := range d.DceDeps {
			o := string(dep)
			if decls, ok := declsByObject[o]; ok {
				delete(declsByObject, o)
				name := strings.Split(o, ":")[1]
				for _, d := range decls {
					for i, f := range d.DceFilters {
						if string(f) == name {
							d.DceFilters[i] = d.DceFilters[len(d.DceFilters)-1]
							d.DceFilters = d.DceFilters[:len(d.DceFilters)-1]
							break
						}
					}
					if len(d.DceFilters) == 0 {
						pendingDecls = append(pendingDecls, d)
					}
				}
			}
		}
	}

	if _, err := w.Write([]byte("\"use strict\";\n(function() {\n\n")); err != nil {
		return err
	}
	if _, err := w.Write(removeWhitespace([]byte(prelude.Prelude), minify)); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}

	// write packages
	for _, pkg := range pkgs {
		if err := WritePkgCode(pkg, minify, w); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte("$go($packages[\"" + string(mainPkg.ImportPath) + "\"].$init, [], true);\n$flushConsole();\n\n})();\n")); err != nil {
		return err
	}

	return nil
}

func WritePkgCode(pkg *Archive, minify bool, w *SourceMapFilter) error {
	if w.MappingCallback != nil && pkg.FileSet != nil {
		w.fileSet = token.NewFileSet()
		if err := w.fileSet.Read(json.NewDecoder(bytes.NewReader(pkg.FileSet)).Decode); err != nil {
			panic(err)
		}
	}
	if _, err := w.Write(removeWhitespace([]byte(fmt.Sprintf("$packages[\"%s\"] = (function() {\n", pkg.ImportPath)), minify)); err != nil {
		return err
	}
	vars := []string{"$pkg = {}"}
	for i := range pkg.Imports {
		vars = append(vars, fmt.Sprintf("%s = $packages[\"%s\"]", pkg.Imports[i].VarName, pkg.Imports[i].Path))
	}
	for i := range pkg.Declarations {
		if len(pkg.Declarations[i].DceFilters) == 0 {
			vars = append(vars, pkg.Declarations[i].Vars...)
		}
	}
	if len(vars) != 0 {
		if _, err := w.Write(removeWhitespace([]byte(fmt.Sprintf("\tvar %s;\n", strings.Join(vars, ", "))), minify)); err != nil {
			return err
		}
	}
	for i := range pkg.Declarations {
		if len(pkg.Declarations[i].DceFilters) == 0 {
			if _, err := w.Write(pkg.Declarations[i].BodyCode); err != nil {
				return err
			}
		}
	}

	if _, err := w.Write(removeWhitespace([]byte("\t$pkg.$init = function() {\n\t\t$pkg.$init = function() {};\n\t\t/* */ var $r, $s = 0; var $f = function() { while (true) { switch ($s) { case 0:\n"), minify)); err != nil {
		return err
	}
	for i := range pkg.Declarations {
		if len(pkg.Declarations[i].DceFilters) == 0 {
			if _, err := w.Write(pkg.Declarations[i].InitCode); err != nil {
				return err
			}
		}
	}
	if _, err := w.Write(removeWhitespace([]byte("\t\t/* */ } return; } }; $f.$blocking = true; return $f;\n\t};\n\treturn $pkg;\n})();"), minify)); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil { // keep this \n even when minified
		return err
	}
	return nil
}

func ReadArchive(filename, id string, r io.Reader, packages map[string]*types.Package) (*Archive, error) {
	var a Archive
	if err := gob.NewDecoder(r).Decode(&a); err != nil {
		return nil, err
	}

	pkg, err := gcimporter.ImportData(packages, filename, id, bytes.NewReader(a.GcData))
	if err != nil {
		return nil, err
	}
	packages[pkg.Path()] = pkg

	return &a, nil
}

func WriteArchive(a *Archive, w io.Writer) error {
	return gob.NewEncoder(w).Encode(a)
}

type Archive struct {
	ImportPath   string
	GcData       []byte
	Imports      []*PkgImport
	Declarations []*Decl
	FileSet      []byte
	Minified     bool
}

type PkgImport struct {
	Path    string
	VarName string
}

type Decl struct {
	FullName   string
	Vars       []string
	BodyCode   []byte
	InitCode   []byte
	DceFilters []string
	DceDeps    []string
	Blocking   bool
}

type SourceMapFilter struct {
	Writer          io.Writer
	MappingCallback func(generatedLine, generatedColumn int, fileSet *token.FileSet, originalPos token.Pos)
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
			f.MappingCallback(f.line+1, f.column, f.fileSet, token.Pos(binary.BigEndian.Uint32(p[i+1:i+5])))
		}
		p = p[i+5:]
		n += 5
	}
}
