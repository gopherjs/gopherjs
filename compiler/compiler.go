package compiler

import (
	"bytes"
	"code.google.com/p/go.tools/go/gcimporter"
	"code.google.com/p/go.tools/go/types"
	"encoding/asn1"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"go/token"
	"io"
	"sort"
	"strings"
)

const GOROUTINES = false

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

type ImportContext struct {
	Packages map[string]*types.Package
	Import   func(string) (*Archive, error)
}

func NewImportContext(importFunc func(string) (*Archive, error)) *ImportContext {
	return &ImportContext{
		Packages: map[string]*types.Package{"unsafe": types.Unsafe},
		Import:   importFunc,
	}
}

func WriteProgramCode(pkgs []*Archive, importContext *ImportContext, w *SourceMapFilter) error {
	mainPkg := pkgs[len(pkgs)-1]
	minify := mainPkg.Minified

	declsByObject := make(map[string][]*Decl)
	var pendingDecls []*Decl
	for _, pkg := range pkgs {
		for i := range pkg.Declarations {
			d := &pkg.Declarations[i]
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
	if _, err := w.Write(removeWhitespace(prelude, minify)); err != nil {
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

	// write interfaces
	allTypeNames := []*types.TypeName{types.New("error").(*types.Named).Obj()}
	for _, pkg := range pkgs {
		scope := importContext.Packages[string(pkg.ImportPath)].Scope()
		for _, name := range scope.Names() {
			if typeName, isTypeName := scope.Lookup(name).(*types.TypeName); isTypeName {
				if _, notUsed := declsByObject[string(pkg.ImportPath)+":"+name]; !notUsed {
					allTypeNames = append(allTypeNames, typeName)
				}
			}
		}
	}
	for _, t := range allTypeNames {
		if in, isInterface := t.Type().Underlying().(*types.Interface); isInterface {
			if in.Empty() {
				continue
			}
			implementedBy := make(map[string]bool, 0)
			for _, other := range allTypeNames {
				otherType := other.Type()
				switch otherType.Underlying().(type) {
				case *types.Interface:
					// skip
				case *types.Struct:
					if types.AssignableTo(otherType, in) {
						implementedBy[fmt.Sprintf("$packages[\"%s\"].%s", other.Pkg().Path(), other.Name())] = true
					}
					if types.AssignableTo(types.NewPointer(otherType), in) {
						implementedBy[fmt.Sprintf("$packages[\"%s\"].%s.Ptr", other.Pkg().Path(), other.Name())] = true
					}
				default:
					if types.AssignableTo(otherType, in) {
						implementedBy[fmt.Sprintf("$packages[\"%s\"].%s", other.Pkg().Path(), other.Name())] = true
					}
					if types.AssignableTo(types.NewPointer(otherType), in) {
						implementedBy[fmt.Sprintf("$ptrType($packages[\"%s\"].%s)", other.Pkg().Path(), other.Name())] = true
					}
				}
			}
			list := make([]string, 0, len(implementedBy))
			for ref := range implementedBy {
				list = append(list, ref)
			}
			sort.Strings(list)
			var target string
			switch t.Name() {
			case "error":
				target = "$error"
			default:
				target = fmt.Sprintf("$packages[\"%s\"].%s", t.Pkg().Path(), t.Name())
			}
			if _, err := w.Write(removeWhitespace([]byte(fmt.Sprintf("%s.implementedBy = [%s];\n", target, strings.Join(list, ", "))), minify)); err != nil {
				return err
			}
		}
	}

	for _, pkg := range pkgs {
		if _, err := w.Write([]byte("$packages[\"" + string(pkg.ImportPath) + "\"].$init();\n")); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte("$packages[\"" + string(mainPkg.ImportPath) + "\"].main(function() {});\n\n})();\n")); err != nil {
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
	for _, imp := range pkg.Imports {
		vars = append(vars, fmt.Sprintf("%s = $packages[\"%s\"]", imp.VarName, imp.Path))
	}
	for _, d := range pkg.Declarations {
		if len(d.DceFilters) == 0 && d.Var != "" {
			vars = append(vars, d.Var)
		}
	}
	if len(vars) != 0 {
		if _, err := w.Write(removeWhitespace([]byte(fmt.Sprintf("\tvar %s;\n", strings.Join(vars, ", "))), minify)); err != nil {
			return err
		}
	}
	for _, d := range pkg.Declarations {
		if len(d.DceFilters) == 0 {
			if _, err := w.Write(d.BodyCode); err != nil {
				return err
			}
		}
	}
	if _, err := w.Write(removeWhitespace([]byte("\t$pkg.$init = function() {\n"), minify)); err != nil {
		return err
	}
	for _, d := range pkg.Declarations {
		if len(d.DceFilters) == 0 {
			if _, err := w.Write(d.InitCode); err != nil {
				return err
			}
		}
	}
	if _, err := w.Write(removeWhitespace([]byte("\t};\n\treturn $pkg;\n})();"), minify)); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil { // keep this \n even when minified
		return err
	}
	return nil
}

func UnmarshalArchive(filename, id string, data []byte, importContext *ImportContext) (*Archive, error) {
	var a Archive
	_, err := asn1.Unmarshal(data, &a)
	if err != nil {
		return nil, err
	}

	pkg, err := gcimporter.ImportData(importContext.Packages, filename, id, bytes.NewReader(a.GcData))
	if err != nil {
		return nil, err
	}
	importContext.Packages[pkg.Path()] = pkg

	return &a, nil
}

func MarshalArchive(a *Archive) ([]byte, error) {
	return asn1.Marshal(*a)
}

type Archive struct {
	ImportPath   PkgPath
	GcData       []byte
	Dependencies []PkgPath
	Imports      []PkgImport
	Declarations []Decl
	Tests        []string
	FileSet      []byte
	Minified     bool
}

type PkgPath []byte // make asn1 happy

func (a *Archive) AddDependency(path string) {
	for _, dep := range a.Dependencies {
		if string(dep) == path {
			return
		}
	}
	a.Dependencies = append(a.Dependencies, PkgPath(path))
}

func (a *Archive) AddDependenciesOf(other *Archive) {
	for _, path := range other.Dependencies {
		a.AddDependency(string(path))
	}
	a.AddDependency(string(other.ImportPath))
}

type PkgImport struct {
	Path    PkgPath
	VarName string
}

type Decl struct {
	Var        string
	BodyCode   []byte
	InitCode   []byte
	DceFilters []DepId
	DceDeps    []DepId
}

type DepId []byte // make asn1 happy

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
