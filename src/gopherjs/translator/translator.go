package translator

import (
	"bufio"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"strings"
	"time"
)

type Translator struct {
	BuildContext *build.Context
	TypesConfig  *types.Config
	GetModTime   func(string) time.Time
	StorePackage func(*GopherPackage) error
	FileSet      *token.FileSet
	Packages     map[string]*GopherPackage
}

type GopherPackage struct {
	*build.Package
	SrcModTime     time.Time
	JavaScriptCode []byte
}

func (t *Translator) getPackage(importPath string, srcDir string) (*GopherPackage, error) {
	if pkg, found := t.Packages[importPath]; found {
		return pkg, nil
	}

	otherPkg, err := t.BuildContext.Import(importPath, srcDir, build.AllowBinary)
	if err != nil {
		return nil, err
	}
	pkg := &GopherPackage{Package: otherPkg}
	t.Packages[importPath] = pkg
	if err := t.BuildPackage(pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (t *Translator) BuildPackage(pkg *GopherPackage) error {
	if pkg.ImportPath == "unsafe" {
		t.TypesConfig.Packages["unsafe"] = types.Unsafe
		return nil
	}

	pkg.SrcModTime = t.GetModTime("") // gopherjs itself

	for _, importedPkgPath := range pkg.Imports {
		compiledPkg, err := t.getPackage(importedPkgPath, pkg.Dir)
		if err != nil {
			return err
		}
		if compiledPkg.SrcModTime.After(pkg.SrcModTime) {
			pkg.SrcModTime = compiledPkg.SrcModTime
		}
	}

	for _, name := range pkg.GoFiles {
		fileModTime := t.GetModTime(pkg.Dir + "/" + name)
		if fileModTime.After(pkg.SrcModTime) {
			pkg.SrcModTime = fileModTime
		}
	}

	pkgObjModTime := t.GetModTime(pkg.PkgObj)
	if pkgObjModTime.Unix() != 0 && !pkg.SrcModTime.After(pkgObjModTime) && pkg.PkgObj != "" {
		// package object is up to date, load from disk if library
		if pkg.IsCommand() {
			return nil
		}

		objFile, err := t.BuildContext.OpenFile(pkg.PkgObj)
		if err != nil {
			return err
		}
		defer objFile.Close()

		r := bufio.NewReader(objFile)
		for {
			line, err := r.ReadSlice('\n')
			if err != nil && err != bufio.ErrBufferFull {
				return err
			}
			if len(line) == 3 && string(line) == "$$\n" {
				break
			}
			pkg.JavaScriptCode = append(pkg.JavaScriptCode, line...)
		}

		t.TypesConfig.Packages[pkg.ImportPath], err = types.GcImportData(t.TypesConfig.Packages, pkg.PkgObj, pkg.ImportPath, r)
		return err
	}

	files := make([]*ast.File, 0)
	for _, name := range pkg.GoFiles {
		fullName := pkg.Dir + "/" + name
		r, err := t.BuildContext.OpenFile(fullName)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(t.FileSet, fullName, r, 0)
		if err != nil {
			return err
		}
		files = append(files, file)
	}

	t.TypesConfig.Import = func(imports map[string]*types.Package, path string) (*types.Package, error) {
		_, err := t.getPackage(path, pkg.Dir)
		if err != nil {
			return nil, err
		}
		return imports[path], nil
	}
	packageCode, err := translatePackage(pkg.ImportPath, files, t.FileSet, t.TypesConfig)
	if err != nil {
		return err
	}

	var jsCode []byte
	if pkg.IsCommand() {
		jsCode = []byte(strings.TrimSpace(prelude))
		jsCode = append(jsCode, '\n')

		loaded := make(map[*types.Package]bool)
		var loadImportsOf func(*GopherPackage) error
		loadImportsOf = func(of *GopherPackage) error {
			for _, imp := range t.TypesConfig.Packages[of.ImportPath].Imports() {
				if imp.Path() == "unsafe" || imp.Path() == "reflect" || imp.Path() == "go/doc" {
					continue
				}
				if _, alreadyLoaded := loaded[imp]; alreadyLoaded {
					continue
				}
				loaded[imp] = true

				gopherPkg, err := t.getPackage(imp.Path(), pkg.Dir)
				if err != nil {
					return err
				}

				if err := loadImportsOf(gopherPkg); err != nil {
					return err
				}

				jsCode = append(jsCode, []byte(`Go$packages["`+imp.Path()+`"] = (function() {`)...)
				jsCode = append(jsCode, gopherPkg.JavaScriptCode...)
				exports := make([]string, 0)
				for _, name := range imp.Scope().Names() {
					if ast.IsExported(name) {
						exports = append(exports, fmt.Sprintf("%s: %s", name, name))
					}
				}
				jsCode = append(jsCode, []byte("\treturn { "+strings.Join(exports, ", ")+" };\n")...)
				jsCode = append(jsCode, []byte("})();\n")...)
			}
			return nil
		}
		if err := loadImportsOf(pkg); err != nil {
			return err
		}
	}
	jsCode = append(jsCode, packageCode...)
	if pkg.IsCommand() {
		jsCode = append(jsCode, []byte("main();\n")...)
	}
	pkg.JavaScriptCode = jsCode

	if pkg.PkgObj != "" {
		return t.StorePackage(pkg)
	}

	return nil
}
