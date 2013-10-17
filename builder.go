package main

import (
	"bytes"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"github.com/neelance/gopherjs/translator"
	"go/ast"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path"
	"strings"
	"time"
)

type Builder struct {
	BuildContext *build.Context
	TypesConfig  *types.Config
	FileSet      *token.FileSet
	Packages     map[string]*BuilderPackage
}

type BuilderPackage struct {
	*build.Package
	SrcModTime     time.Time
	JavaScriptCode []byte
}

var PkgObjUpToDate = fmt.Errorf("Package object already up-to-date.")

func (b *Builder) BuildPackage(pkg *BuilderPackage) error {
	if pkg.ImportPath == "unsafe" {
		b.TypesConfig.Packages["unsafe"] = types.Unsafe
		return nil
	}

	b.TypesConfig.Import = func(imports map[string]*types.Package, path string) (*types.Package, error) {
		if _, found := b.Packages[path]; found {
			return imports[path], nil
		}

		otherPkg, err := b.BuildContext.Import(path, pkg.Dir, build.AllowBinary)
		if err != nil {
			return nil, err
		}
		pkg := &BuilderPackage{Package: otherPkg}
		b.Packages[path] = pkg
		if err := b.BuildPackage(pkg); err != nil && err != PkgObjUpToDate {
			return nil, err
		}

		return imports[path], nil
	}

	fileInfo, err := os.Stat(os.Args[0]) // gopherjs itself
	if err != nil {
		return err
	}
	pkg.SrcModTime = fileInfo.ModTime()

	for _, importedPkgPath := range pkg.Imports {
		_, err := b.TypesConfig.Import(b.TypesConfig.Packages, importedPkgPath)
		if err != nil {
			return err
		}
		impModeTime := b.Packages[importedPkgPath].SrcModTime
		if impModeTime.After(pkg.SrcModTime) {
			pkg.SrcModTime = impModeTime
		}
	}

	for _, name := range pkg.GoFiles {
		fileInfo, err := os.Stat(pkg.Dir + "/" + name)
		if err != nil {
			return err
		}
		if fileInfo.ModTime().After(pkg.SrcModTime) {
			pkg.SrcModTime = fileInfo.ModTime()
		}
	}

	pkgObjFileInfo, err := os.Stat(pkg.PkgObj)
	if err != nil && !pkg.SrcModTime.After(pkgObjFileInfo.ModTime()) && pkg.PkgObj != "" {
		// package object is up to date, load from disk if library
		if pkg.IsCommand() {
			return PkgObjUpToDate
		}

		objFile, err := os.Open(pkg.PkgObj)
		if err != nil {
			return err
		}
		defer objFile.Close()

		pkg.JavaScriptCode, _, err = translator.ReadArchive(b.TypesConfig.Packages, pkg.PkgObj, pkg.ImportPath, objFile)
		if err != nil {
			return err
		}

		return PkgObjUpToDate
	}

	files := make([]*ast.File, 0)
	var errList translator.ErrorList
	for _, name := range pkg.GoFiles {
		fullName := pkg.Dir + "/" + name
		r, err := os.Open(fullName)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(b.FileSet, fullName, r, 0)
		r.Close()
		if err != nil {
			if list, isList := err.(scanner.ErrorList); isList {
				for _, entry := range list {
					errList = append(errList, entry)
				}
				continue
			}
			errList = append(errList, err)
			continue
		}
		files = append(files, file)
	}
	if errList != nil {
		return errList
	}

	pkg.JavaScriptCode, err = translator.TranslatePackage(pkg.ImportPath, files, b.FileSet, b.TypesConfig)
	if err != nil {
		return err
	}

	if !pkg.IsCommand() {
		if err := os.MkdirAll(path.Dir(pkg.PkgObj), 0777); err != nil {
			return err
		}
		file, err := os.Create(pkg.PkgObj)
		if err != nil {
			return err
		}
		translator.WriteArchive(pkg.JavaScriptCode, b.TypesConfig.Packages[pkg.ImportPath], file)
		file.Close()
		return nil
	}

	b.Packages[pkg.ImportPath] = pkg
	dependencies, err := translator.GetAllDependencies(pkg.ImportPath, b.TypesConfig)
	if err != nil {
		return err
	}

	jsCode := bytes.NewBuffer(nil)
	jsCode.WriteString(strings.TrimSpace(translator.Prelude))
	jsCode.WriteRune('\n')

	for _, dep := range dependencies {
		jsCode.WriteString("Go$packages[\"" + dep.Path() + "\"] = (function() {\n")
		jsCode.Write(b.Packages[dep.Path()].JavaScriptCode)
		jsCode.WriteString("})();\n")
	}

	translator.WriteInterfaces(dependencies, jsCode, false)

	for _, dep := range dependencies {
		if dep.Scope().Lookup("init") != nil {
			jsCode.WriteString("Go$packages[\"" + dep.Path() + "\"].init();\n")
		}
	}

	jsCode.WriteString("Go$packages[\"" + pkg.ImportPath + "\"].main();\n")

	pkg.JavaScriptCode = jsCode.Bytes()

	return nil
}
