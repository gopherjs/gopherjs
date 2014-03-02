package translator

import (
	"bytes"
	"code.google.com/p/go.tools/go/gcimporter"
	"code.google.com/p/go.tools/go/types"
	"encoding/asn1"
	"fmt"
	"go/ast"
	"io"
	"sort"
	"strings"
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

type Translator struct {
	typesPackages map[string]*types.Package
}

func New() *Translator {
	return &Translator{map[string]*types.Package{"unsafe": types.Unsafe}}
}

func (t *Translator) NewEmptyTypesPackage(path string) {
	t.typesPackages[path] = types.NewPackage(path, path)
}

func (t *Translator) WriteProgramCode(pkgs []*Archive, mainPkgPath string, w io.Writer) {
	declsByObject := make(map[Object][]*Decl)
	var pendingDecls []*Decl
	for _, pkg := range pkgs {
		for i := range pkg.Declarations {
			d := &pkg.Declarations[i]
			if len(d.DceFilters) == 0 {
				pendingDecls = append(pendingDecls, d)
				continue
			}
			for _, f := range d.DceFilters {
				o := Object{pkg.ImportPath, f}
				declsByObject[o] = append(declsByObject[o], d)
			}
		}
	}

	for len(pendingDecls) != 0 {
		d := pendingDecls[len(pendingDecls)-1]
		pendingDecls = pendingDecls[:len(pendingDecls)-1]
		for _, o := range d.DceDeps {
			if decls, ok := declsByObject[o]; ok {
				delete(declsByObject, o)
				for _, d := range decls {
					for i, f := range d.DceFilters {
						if f == o.Name {
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

	w.Write([]byte("\"use strict\";\n(function() {\n\n"))
	w.Write([]byte(strings.TrimSpace(prelude)))
	w.Write([]byte("\n"))

	// write packages
	for _, pkg := range pkgs {
		WritePkgCode(pkg, w)
	}

	// write interfaces
	allTypeNames := []*types.TypeName{types.New("error").(*types.Named).Obj()}
	for _, pkg := range pkgs {
		scope := t.typesPackages[pkg.ImportPath].Scope()
		for _, name := range scope.Names() {
			if typeName, isTypeName := scope.Lookup(name).(*types.TypeName); isTypeName {
				if _, notUsed := declsByObject[Object{pkg.ImportPath, strings.Replace(name, "_", "-", -1)}]; !notUsed {
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
						implementedBy[fmt.Sprintf("go$packages[\"%s\"].%s", other.Pkg().Path(), other.Name())] = true
					}
					if types.AssignableTo(types.NewPointer(otherType), in) {
						implementedBy[fmt.Sprintf("go$packages[\"%s\"].%s.Ptr", other.Pkg().Path(), other.Name())] = true
					}
				default:
					if types.AssignableTo(otherType, in) {
						implementedBy[fmt.Sprintf("go$packages[\"%s\"].%s", other.Pkg().Path(), other.Name())] = true
					}
					if types.AssignableTo(types.NewPointer(otherType), in) {
						implementedBy[fmt.Sprintf("go$ptrType(go$packages[\"%s\"].%s)", other.Pkg().Path(), other.Name())] = true
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
				target = "go$error"
			default:
				target = fmt.Sprintf("go$packages[\"%s\"].%s", t.Pkg().Path(), t.Name())
			}
			fmt.Fprintf(w, "%s.implementedBy = [%s];\n", target, strings.Join(list, ", "))
		}
	}

	for _, pkg := range pkgs {
		w.Write([]byte("go$packages[\"" + pkg.ImportPath + "\"].init();\n"))
	}

	w.Write([]byte("go$packages[\"" + mainPkgPath + "\"].main();\n\n})();"))
}

func WritePkgCode(pkg *Archive, w io.Writer) {
	fmt.Fprintf(w, "go$packages[\"%s\"] = (function() {\n", pkg.ImportPath)
	vars := []string{"go$pkg = {}"}
	for _, imp := range pkg.Imports {
		vars = append(vars, fmt.Sprintf("%s = go$packages[\"%s\"]", imp.VarName, imp.Path))
	}
	for _, d := range pkg.Declarations {
		if len(d.DceFilters) == 0 && d.Var != "" {
			vars = append(vars, d.Var)
		}
	}
	if len(vars) != 0 {
		fmt.Fprintf(w, "\tvar %s;\n", strings.Join(vars, ", "))
	}
	for _, d := range pkg.Declarations {
		if len(d.DceFilters) == 0 {
			w.Write(d.BodyCode)
		}
	}
	w.Write([]byte("\tgo$pkg.init = function() {\n"))
	for _, d := range pkg.Declarations {
		if len(d.DceFilters) == 0 {
			w.Write(d.InitCode)
		}
	}
	w.Write([]byte("\t}\n\treturn go$pkg;\n})();\n"))
}

func (t *Translator) ReadArchive(filename, id string, data []byte) (*Archive, error) {
	var a Archive
	_, err := asn1.Unmarshal(data, &a)
	if err != nil {
		return nil, err
	}

	pkg, err := gcimporter.ImportData(t.typesPackages, filename, id, bytes.NewReader(a.GcData))
	if err != nil {
		return nil, err
	}
	t.typesPackages[pkg.Path()] = pkg

	return &a, nil
}

func (t *Translator) WriteArchive(a *Archive) ([]byte, error) {
	return asn1.Marshal(*a)
}

type Archive struct {
	ImportPath   string
	GcData       []byte
	Dependencies []string
	Imports      []Import
	Declarations []Decl
	Tests        []string
}

func (a *Archive) AddDependency(path string) {
	for _, dep := range a.Dependencies {
		if dep == path {
			return
		}
	}
	a.Dependencies = append(a.Dependencies, path)
}

func (a *Archive) AddDependenciesOf(other *Archive) {
	for _, path := range other.Dependencies {
		a.AddDependency(path)
	}
}

type Import struct {
	Path    string
	VarName string
}

type Decl struct {
	Var        string
	BodyCode   []byte
	InitCode   []byte
	DceFilters []string
	DceDeps    []Object
}

type Object struct {
	PkgPath string
	Name    string
}

func (c *pkgContext) translateArgs(sig *types.Signature, args []ast.Expr, ellipsis bool) string {
	params := make([]string, sig.Params().Len())
	for i := range params {
		if sig.Variadic() && i == len(params)-1 && !ellipsis {
			varargType := sig.Params().At(i).Type().(*types.Slice)
			varargs := make([]string, len(args)-i)
			for j, arg := range args[i:] {
				varargs[j] = c.translateImplicitConversion(arg, varargType.Elem()).String()
			}
			params[i] = fmt.Sprintf("new %s([%s])", c.typeName(varargType), strings.Join(varargs, ", "))
			break
		}
		argType := sig.Params().At(i).Type()
		params[i] = c.translateImplicitConversion(args[i], argType).String()
	}
	return strings.Join(params, ", ")
}

func (c *pkgContext) translateSelection(sel *types.Selection) (fields []string, jsTag string) {
	t := sel.Recv()
	for _, index := range sel.Index() {
		if ptr, isPtr := t.(*types.Pointer); isPtr {
			t = ptr.Elem()
		}
		s := t.Underlying().(*types.Struct)
		if jsTag = getJsTag(s.Tag(index)); jsTag != "" {
			for i := 0; i < s.NumFields(); i++ {
				if isJsObject(s.Field(i).Type()) {
					fields = append(fields, fieldName(s, i))
					return
				}
			}
		}
		fields = append(fields, fieldName(s, index))
		t = s.Field(index).Type()
	}
	return
}
