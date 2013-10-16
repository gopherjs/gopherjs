package translator

import (
	"bufio"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"github.com/neelance/gopherjs/gcexporter"
	"io"
	"sort"
	"strings"
)

func GetAllDependencies(pkg string, config *types.Config) ([]*types.Package, error) {
	var dependencies []*types.Package // ordered
	imported := make(map[string]bool)
	var importPkg func(string) error
	importPkg = func(importPath string) error {
		if importPath == "unsafe" || importPath == "reflect" || importPath == "go/doc" {
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
	err := importPkg(pkg)
	return dependencies, err
}

func WriteInterfaces(dependencies []*types.Package, w io.Writer) {
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
			if t.Name() == "error" {
				fmt.Fprintf(w, "Go$error.Go$implementedBy = [%s];\n", strings.Join(list, ", "))
				continue
			}
			fmt.Fprintf(w, "Go$packages[\"%s\"].%s.Go$implementedBy = [%s];\n", t.Pkg().Path(), t.Name(), strings.Join(list, ", "))
		}
	}
}

func ReadArchive(imports map[string]*types.Package, filename, id string, data io.Reader) ([]byte, *types.Package, error) {
	r := bufio.NewReader(data)
	code := make([]byte, 0)
	for {
		line, err := r.ReadSlice('\n')
		if err != nil && err != bufio.ErrBufferFull {
			return nil, nil, err
		}
		if len(line) == 3 && string(line) == "$$\n" {
			break
		}
		code = append(code, line...)
	}

	pkg, err := types.GcImportData(imports, filename, id, r)
	if err != nil {
		return nil, nil, err
	}

	return code, pkg, nil
}

func WriteArchive(code []byte, pkg *types.Package, w io.Writer) {
	w.Write(code)
	w.Write([]byte("$$\n"))
	gcexporter.Write(pkg, w)
}
