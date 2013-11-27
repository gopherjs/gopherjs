package translator

import (
	"bufio"
	"code.google.com/p/go.tools/go/gcimporter"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"github.com/neelance/gopherjs/gcexporter"
	"io"
	"sort"
	"strings"
)

var sizes32 = &types.StdSizes{WordSize: 4, MaxAlign: 8}

func GetAllDependencies(pkg string, config *types.Config) ([]*types.Package, error) {
	var dependencies []*types.Package // ordered
	imported := make(map[string]bool)
	var importPkg func(string) error
	importPkg = func(importPath string) error {
		if importPath == "unsafe" || importPath == "go/doc" {
			return nil
		}
		if _, found := imported[importPath]; found {
			return nil
		}
		imported[importPath] = true

		typesPkg, err := config.Import(config.Packages, importPath)
		if err != nil {
			return err
		}
		var imps []string
		for _, imp := range typesPkg.Imports() {
			imps = append(imps, imp.Path())
		}
		sort.Strings(imps)
		for _, imp := range imps {
			if err := importPkg(imp); err != nil {
				return err
			}
		}

		dependencies = append(dependencies, typesPkg)
		return nil
	}
	importPkg("runtime") // all packages depend on runtime
	err := importPkg(pkg)
	return dependencies, err
}

func WriteInterfaces(dependencies []*types.Package, w io.Writer, merge bool) {
	allTypeNames := []*types.TypeName{types.New("error").(*types.Named).Obj()}
	for _, dep := range dependencies {
		scope := dep.Scope()
		for _, name := range scope.Names() {
			if typeName, isTypeName := scope.Lookup(name).(*types.TypeName); isTypeName {
				allTypeNames = append(allTypeNames, typeName)
			}
		}
	}
	for _, t := range allTypeNames {
		if in, isInterface := t.Type().Underlying().(*types.Interface); isInterface {
			if in.MethodSet().Len() == 0 {
				continue
			}
			implementedBy := make(map[string]bool, 0)
			for _, other := range allTypeNames {
				otherType := other.Type()
				if _, otherIsInterface := otherType.Underlying().(*types.Interface); otherIsInterface {
					continue
				}
				if _, isStruct := otherType.Underlying().(*types.Struct); isStruct {
					if types.IsAssignableTo(otherType, in) {
						implementedBy[fmt.Sprintf("Go$packages[\"%s\"].%s.Go$NonPointer", other.Pkg().Path(), other.Name())] = true
					}
					otherType = types.NewPointer(otherType)
				}
				if types.IsAssignableTo(otherType, in) {
					implementedBy[fmt.Sprintf("Go$packages[\"%s\"].%s", other.Pkg().Path(), other.Name())] = true
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
				target = "Go$error"
			default:
				target = fmt.Sprintf("Go$packages[\"%s\"].%s", t.Pkg().Path(), t.Name())
			}
			if merge {
				for _, entry := range list {
					fmt.Fprintf(w, "if (%s.Go$implementedBy.indexOf(%s) === -1) { %s.Go$implementedBy.push(%s); }\n", target, entry, target, entry)
				}
				continue
			}
			fmt.Fprintf(w, "%s.Go$implementedBy = [%s];\n", target, strings.Join(list, ", "))
		}
	}
}

// TODO replace with encoding/gob when reflection is ready
func ReadArchive(packages map[string]*types.Package, filename, id string, data io.Reader) ([]byte, *types.Package, error) {
	r := bufio.NewReader(data)

	code, err := readUntilSeparator(r)
	if err != nil {
		return nil, nil, err
	}
	importList, err := readUntilSeparator(r)
	if err != nil {
		return nil, nil, err
	}

	pkg, err := gcimporter.ImportData(packages, filename, id, r)
	if err != nil {
		return nil, nil, err
	}

	var imports []*types.Package
	for _, path := range strings.Split(string(importList), "\n") {
		if path == "" {
			continue
		}
		impPkg, found := packages[path]
		if !found {
			impPkg = types.NewPackage(path, "", types.NewScope(nil))
			packages[path] = impPkg
		}
		imports = append(imports, impPkg)
	}
	pkg.SetImports(imports)

	return code, pkg, nil
}

func readUntilSeparator(r *bufio.Reader) ([]byte, error) {
	var content []byte
	for {
		line, err := r.ReadSlice('\n')
		if err != nil && err != bufio.ErrBufferFull {
			return nil, err
		}
		if len(line) == 3 && string(line) == "$$\n" {
			break
		}
		content = append(content, line...)
	}
	return content, nil
}

func WriteArchive(code []byte, pkg *types.Package, w io.Writer) {
	w.Write(code)
	w.Write([]byte("$$\n"))
	for _, impPkg := range pkg.Imports() {
		w.Write([]byte(impPkg.Path()))
		w.Write([]byte{'\n'})
	}
	w.Write([]byte("$$\n"))
	gcexporter.Write(pkg, w, sizes32)
}
