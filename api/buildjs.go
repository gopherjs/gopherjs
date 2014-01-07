package api

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"time"

	"code.google.com/p/go.tools/go/types"
	"github.com/neelance/gopherjs/translator"
)

type Package struct {
	*build.Package
	SrcModTime     time.Time
	UpToDate       bool
	JavaScriptCode []byte
}

var fileSet = token.NewFileSet()
var packages = make(map[string]*Package)

var TypesConfig = &types.Config{
	Packages: make(map[string]*types.Package),
}

func SetPackage(key string, pkg *Package) {
	packages[key] = pkg
}

var InstallMode = false

func init() {
	TypesConfig.Import = func(imports map[string]*types.Package, path string) (*types.Package, error) {
		if _, found := packages[path]; found {
			return imports[path], nil
		}

		otherPkg, err := BuildImport(path, build.AllowBinary)
		if err != nil {
			return nil, err
		}
		pkg := &Package{Package: otherPkg}
		if err := BuildPackage(pkg); err != nil {
			return nil, err
		}

		return imports[path], nil
	}
}

func BuildImport(path string, mode build.ImportMode) (*build.Package, error) {
	buildContext := &build.Context{
		GOROOT:   build.Default.GOROOT,
		GOPATH:   build.Default.GOPATH,
		GOOS:     "darwin",
		GOARCH:   "js",
		Compiler: "gc",
	}
	if path == "runtime" || path == "syscall" {
		buildContext.GOARCH = build.Default.GOARCH
		buildContext.InstallSuffix = "js"
	}
	pkg, err := buildContext.Import(path, "", mode)
	if path == "hash/crc32" {
		pkg.GoFiles = []string{"crc32.go", "crc32_generic.go"}
	}
	if _, err := os.Stat(pkg.PkgObj); os.IsNotExist(err) && strings.HasPrefix(pkg.PkgObj, build.Default.GOROOT) {
		// fall back to GOPATH
		gopathPkgObj := build.Default.GOPATH + pkg.PkgObj[len(build.Default.GOROOT):]
		if _, err := os.Stat(gopathPkgObj); err == nil {
			pkg.PkgObj = gopathPkgObj
		}
	}
	return pkg, err
}

func BuildFiles(filenames []string, pkgObj string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	pkg := &Package{
		Package: &build.Package{
			Name:       "main",
			ImportPath: "main",
			Dir:        wd,
			GoFiles:    filenames,
		},
	}

	if err := BuildPackage(pkg); err != nil {
		return err
	}
	return WriteCommandPackage(pkg, pkgObj)
}

func BuildPackage(pkg *Package) error {
	packages[pkg.ImportPath] = pkg
	if pkg.ImportPath == "unsafe" {
		TypesConfig.Packages["unsafe"] = types.Unsafe
		return nil
	}

	if pkg.PkgObj != "" {
		fileInfo, err := os.Stat(os.Args[0]) // gopherjs itself
		if err != nil {
			for _, path := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
				fileInfo, err = os.Stat(filepath.Join(path, os.Args[0]))
				if err == nil {
					break
				}
			}
			if err != nil {
				os.Stderr.WriteString("Could not get GopherJS binary's modification timestamp. Please report issue.\n")
			}
		}
		if err == nil {
			pkg.SrcModTime = fileInfo.ModTime()
		}

		for _, importedPkgPath := range pkg.Imports {
			_, err := TypesConfig.Import(TypesConfig.Packages, importedPkgPath)
			if err != nil {
				return err
			}
			impModeTime := packages[importedPkgPath].SrcModTime
			if impModeTime.After(pkg.SrcModTime) {
				pkg.SrcModTime = impModeTime
			}
		}

		for _, name := range pkg.GoFiles {
			fileInfo, err := os.Stat(filepath.Join(pkg.Dir, name))
			if err != nil {
				return err
			}
			if fileInfo.ModTime().After(pkg.SrcModTime) {
				pkg.SrcModTime = fileInfo.ModTime()
			}
		}

		pkgObjFileInfo, err := os.Stat(pkg.PkgObj)
		if err == nil && !pkg.SrcModTime.After(pkgObjFileInfo.ModTime()) {
			// package object is up to date, load from disk if library
			pkg.UpToDate = true
			if pkg.IsCommand() {
				return nil
			}

			objFile, err := os.Open(pkg.PkgObj)
			if err != nil {
				return err
			}
			defer objFile.Close()

			pkg.JavaScriptCode, _, err = translator.ReadArchive(TypesConfig.Packages, pkg.PkgObj, pkg.ImportPath, objFile)
			if err != nil {
				return err
			}

			return nil
		}
	}

	wd, _ := os.Getwd()
	files := make([]*ast.File, 0)
	var errList translator.ErrorList
	for _, name := range pkg.GoFiles {
		if pkg.ImportPath == "runtime" && strings.HasPrefix(name, "zgo") {
			continue
		}
		if !filepath.IsAbs(name) {
			name = filepath.Join(pkg.Dir, name)
		}
		r, err := os.Open(name)
		if err != nil {
			return err
		}
		if relname, err := filepath.Rel(wd, name); err == nil {
			name = relname
			if name[0] != '.' {
				name = "." + string(filepath.Separator) + name
			}
		}
		file, err := parser.ParseFile(fileSet, name, r, 0)
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
	if pkg.ImportPath == "runtime" {
		goosFile, _ := parser.ParseFile(fileSet, "zgoos_darwin.go", "package runtime\nconst theGoos = `darwin`\n", 0)
		goarchFile, _ := parser.ParseFile(fileSet, "zgoarch_js.go", "package runtime\nconst theGoarch = `js`\n", 0)
		files = append(files, goosFile, goarchFile)
	}
	if errList != nil {
		return errList
	}

	var err error
	pkg.JavaScriptCode, err = translator.TranslatePackage(pkg.ImportPath, files, fileSet, TypesConfig)
	if err != nil {
		return err
	}

	if InstallMode && !pkg.IsCommand() {
		if err := WriteLibraryPackage(pkg, pkg.PkgObj); err != nil {
			if strings.HasPrefix(pkg.PkgObj, build.Default.GOROOT) {
				// fall back to GOPATH
				if err := WriteLibraryPackage(pkg, build.Default.GOPATH+pkg.PkgObj[len(build.Default.GOROOT):]); err != nil {
					return err
				}
				return nil
			}
			return err
		}
	}

	return nil
}

func WriteLibraryPackage(pkg *Package, pkgObj string) error {
	if err := os.MkdirAll(filepath.Dir(pkgObj), 0777); err != nil {
		return err
	}
	file, err := os.Create(pkgObj)
	if err != nil {
		return err
	}
	defer file.Close()
	translator.WriteArchive(pkg.JavaScriptCode, TypesConfig.Packages[pkg.ImportPath], file)
	return nil
}

func WriteCommandPackage(pkg *Package, pkgObj string) error {
	if pkg.UpToDate {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(pkgObj), 0777); err != nil {
		return err
	}
	file, err := os.Create(pkgObj)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, `"use strict";`)
	file.WriteString(strings.TrimSpace(translator.Prelude))
	file.WriteString("\n")

	dependencies, err := translator.GetAllDependencies(pkg.ImportPath, TypesConfig)
	if err != nil {
		return err
	}

	for _, dep := range dependencies {
		file.WriteString("go$packages[\"" + dep.Path() + "\"] = (function() {\n  var go$pkg = {};\n")
		file.Write(packages[dep.Path()].JavaScriptCode)
		file.WriteString("  return go$pkg;\n})();\n")
	}

	translator.WriteInterfaces(dependencies, file, false)

	for _, dep := range dependencies {
		file.WriteString("go$packages[\"" + dep.Path() + "\"].init();\n")
	}
	file.WriteString("go$packages[\"" + pkg.ImportPath + "\"].main();\n")

	return nil
}
