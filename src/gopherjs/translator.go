package main

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
	buildContext *build.Context
	typesConfig  *types.Config
	getModTime   func(string) time.Time
	storePackage func(*GopherPackage) error
	fileSet      *token.FileSet
	packages     map[string]*GopherPackage
}

type GopherPackage struct {
	*build.Package
	SrcModTime     time.Time
	JavaScriptCode []byte
}

func (t *Translator) getPackage(importPath string, srcDir string) (*GopherPackage, error) {
	if pkg, found := t.packages[importPath]; found {
		return pkg, nil
	}

	otherPkg, err := t.buildContext.Import(importPath, srcDir, build.AllowBinary)
	if err != nil {
		return nil, err
	}
	pkg := &GopherPackage{Package: otherPkg}
	t.packages[importPath] = pkg
	if err := t.buildPackage(pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

func (t *Translator) buildPackage(pkg *GopherPackage) error {
	if pkg.ImportPath == "unsafe" {
		t.typesConfig.Packages["unsafe"] = types.Unsafe
		return nil
	}

	pkg.SrcModTime = t.getModTime("") // gopherjs itself

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
		fileModTime := t.getModTime(pkg.Dir + "/" + name)
		if fileModTime.After(pkg.SrcModTime) {
			pkg.SrcModTime = fileModTime
		}
	}

	pkgObjModTime := t.getModTime(pkg.PkgObj)
	if pkgObjModTime.Unix() != 0 && !pkg.SrcModTime.After(pkgObjModTime) && pkg.PkgObj != "" {
		// package object is up to date, load from disk if library
		if pkg.IsCommand() {
			return nil
		}

		objFile, err := t.buildContext.OpenFile(pkg.PkgObj)
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

		t.typesConfig.Packages[pkg.ImportPath], err = types.GcImportData(t.typesConfig.Packages, pkg.PkgObj, pkg.ImportPath, r)
		return err
	}

	files := make([]*ast.File, 0)
	for _, name := range pkg.GoFiles {
		fullName := pkg.Dir + "/" + name
		r, err := t.buildContext.OpenFile(fullName)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(t.fileSet, fullName, r, 0)
		if err != nil {
			return err
		}
		files = append(files, file)
	}

	packageCode, err := translatePackage(pkg.ImportPath, files, t.fileSet, t.typesConfig)
	if err != nil {
		return err
	}

	var jsCode []byte
	if pkg.IsCommand() {
		jsCode = []byte(strings.TrimSpace(prelude))
		jsCode = append(jsCode, '\n')

		loaded := make(map[*types.Package]bool)
		var loadImportsOf func(*types.Package) error
		loadImportsOf = func(typesPkg *types.Package) error {
			for _, imp := range typesPkg.Imports() {
				if imp.Path() == "unsafe" || imp.Path() == "reflect" || imp.Path() == "go/doc" {
					continue
				}
				if _, alreadyLoaded := loaded[imp]; alreadyLoaded {
					continue
				}
				loaded[imp] = true

				if err := loadImportsOf(imp); err != nil {
					return err
				}

				gopherPkg, err := t.getPackage(imp.Path(), pkg.Dir)
				if err != nil {
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
		if err := loadImportsOf(t.typesConfig.Packages[pkg.ImportPath]); err != nil {
			return err
		}
	}
	jsCode = append(jsCode, packageCode...)
	if pkg.IsCommand() {
		jsCode = append(jsCode, []byte("main();")...)
	}
	pkg.JavaScriptCode = jsCode

	if pkg.PkgObj != "" {
		return t.storePackage(pkg)
	}

	return nil
}
