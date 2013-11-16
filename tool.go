package main

import (
	"code.google.com/p/go.tools/go/exact"
	"code.google.com/p/go.tools/go/types"
	"flag"
	"fmt"
	"github.com/neelance/gopherjs/translator"
	"go/ast"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
	"time"
)

type Package struct {
	*build.Package
	SrcModTime     time.Time
	JavaScriptCode []byte
}

var BuildContext = &build.Context{
	GOROOT:        build.Default.GOROOT,
	GOPATH:        build.Default.GOPATH,
	GOOS:          build.Default.GOOS,
	GOARCH:        build.Default.GOARCH,
	Compiler:      "gc",
	InstallSuffix: "js",
}
var TypesConfig = &types.Config{
	Packages: make(map[string]*types.Package),
}
var FileSet = token.NewFileSet()
var Packages = make(map[string]*Package)
var InstallMode = false

func main() {
	flag.Parse()

	cmd := flag.Arg(0)
	switch cmd {
	case "build":
		buildFlags := flag.NewFlagSet("build", flag.ContinueOnError)
		var pkgObj string
		buildFlags.StringVar(&pkgObj, "o", "", "")
		buildFlags.Parse(flag.Args()[1:])

		if pkgObj == "" {
			basename := path.Base(buildFlags.Arg(0))
			pkgObj = basename[:len(basename)-3] + ".js"
		}
		err := Build(buildFlags.Args(), pkgObj)
		HandleError(err)
		os.Exit(0)

	case "install":
		for _, pkgPath := range flag.Args()[1:] {
			err := Install(pkgPath)
			HandleError(err)
		}
		os.Exit(0)

	case "run":
		tempfile, err := ioutil.TempFile("", path.Base(flag.Arg(1))+".")
		HandleError(err)
		defer func() {
			tempfile.Close()
			os.Remove(tempfile.Name())
		}()
		err = Build(flag.Args()[1:], tempfile.Name())
		HandleError(err)

		node := exec.Command("node", append([]string{tempfile.Name()}, flag.Args()[2:]...)...)
		node.Stdin = os.Stdin
		node.Stdout = os.Stdout
		node.Stderr = os.Stderr
		if err = node.Run(); err != nil {
			if e, isExitError := err.(*exec.ExitError); isExitError {
				os.Exit(e.Sys().(syscall.WaitStatus).ExitStatus())
			}
			HandleError(err)
		}
		os.Exit(0)

	case "tool":
		tool := flag.Arg(1)
		toolFlags := flag.NewFlagSet("tool", flag.ContinueOnError)
		toolFlags.Bool("e", false, "")
		toolFlags.Bool("l", false, "")
		toolFlags.Bool("m", false, "")
		toolFlags.String("o", "", "")
		toolFlags.String("D", "", "")
		toolFlags.String("I", "", "")
		toolFlags.Parse(flag.Args()[2:])

		if len(tool) == 2 {
			switch tool[1] {
			case 'g':
				basename := path.Base(toolFlags.Arg(0))
				err := Build([]string{toolFlags.Arg(0)}, basename[:len(basename)-3]+".js")
				HandleError(err)
				os.Exit(0)
			}
		}
		fmt.Fprintln(os.Stderr, "Tool not supported: "+tool)
		os.Exit(1)

	case "help", "":
		os.Stderr.WriteString(`GopherJS is a tool for compiling Go source code to JavaScript.

Usage:

    gopherjs command [arguments]

The commands are:

    build       compile packages and dependencies
    install     compile and install packages and dependencies
    run         compile and run Go program

`)
		os.Exit(0)

	default:
		fmt.Fprintf(os.Stderr, "gopherjs: unknown subcommand \"%s\"\nRun 'gopherjs help' for usage.\n", cmd)
		os.Exit(1)
	}
}

func HandleError(err error) {
	if err == nil {
		return
	}
	if list, isList := err.(translator.ErrorList); isList {
		for _, entry := range list {
			fmt.Fprintln(os.Stderr, entry)
		}
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}

func Build(filenames []string, pkgObj string) error {
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
			PkgObj:     pkgObj,
		},
	}
	return BuildPackage(pkg)
}

func Install(pkgPath string) error {
	buildPkg, err := BuildContext.Import(pkgPath, "", 0)
	if err != nil {
		return err
	}
	pkg := &Package{Package: buildPkg}
	if pkg.IsCommand() {
		pkg.PkgObj = pkg.BinDir + "/" + path.Base(pkg.ImportPath) + ".js"
	}
	InstallMode = true
	return BuildPackage(pkg)
}

func BuildPackage(pkg *Package) error {
	if pkg.ImportPath == "unsafe" {
		TypesConfig.Packages["unsafe"] = types.Unsafe
		return nil
	}

	TypesConfig.Import = func(imports map[string]*types.Package, path string) (*types.Package, error) {
		if _, found := Packages[path]; found {
			return imports[path], nil
		}

		otherPkg, err := BuildContext.Import(path, pkg.Dir, build.AllowBinary)
		if err != nil {
			return nil, err
		}
		pkg := &Package{Package: otherPkg}
		Packages[path] = pkg
		if err := BuildPackage(pkg); err != nil {
			return nil, err
		}

		return imports[path], nil
	}

	if InstallMode {
		if fileInfo, err := os.Stat(os.Args[0]); err == nil { // gopherjs itself
			pkg.SrcModTime = fileInfo.ModTime()
		}

		for _, importedPkgPath := range pkg.Imports {
			_, err := TypesConfig.Import(TypesConfig.Packages, importedPkgPath)
			if err != nil {
				return err
			}
			impModeTime := Packages[importedPkgPath].SrcModTime
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
		if err == nil && !pkg.SrcModTime.After(pkgObjFileInfo.ModTime()) {
			// package object is up to date, load from disk if library
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

	files := make([]*ast.File, 0)
	var errList translator.ErrorList
	for _, name := range pkg.GoFiles {
		if !path.IsAbs(name) {
			name = path.Join(pkg.Dir, name)
		}
		r, err := os.Open(name)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(FileSet, name, r, 0)
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

	var err error
	pkg.JavaScriptCode, err = translator.TranslatePackage(pkg.ImportPath, files, FileSet, TypesConfig)
	if err != nil {
		return err
	}

	if !pkg.IsCommand() {
		if InstallMode {
			if err := os.MkdirAll(path.Dir(pkg.PkgObj), 0777); err != nil {
				return err
			}
			file, err := os.Create(pkg.PkgObj)
			if err != nil {
				return err
			}
			defer file.Close()
			translator.WriteArchive(pkg.JavaScriptCode, TypesConfig.Packages[pkg.ImportPath], file)
		}
		return nil
	}

	webMode := false
	webModeConst := TypesConfig.Packages[pkg.ImportPath].Scope().Lookup("gopherjsWebMode")
	if webModeConst != nil {
		webMode = exact.BoolVal(webModeConst.(*types.Const).Val())
	}

	if err := os.MkdirAll(path.Dir(pkg.PkgObj), 0777); err != nil {
		return err
	}
	var perm os.FileMode = 0666
	if !webMode {
		perm = 0777
	}
	file, err := os.OpenFile(pkg.PkgObj, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer file.Close()

	if !webMode {
		fmt.Fprintln(file, "#!/usr/bin/env node")
	}
	fmt.Fprintln(file, `"use strict";`)
	fmt.Fprintf(file, "var Go$webMode = %t;\n", webMode)
	file.WriteString(strings.TrimSpace(translator.Prelude))
	file.WriteString("\n")

	Packages[pkg.ImportPath] = pkg
	dependencies, err := translator.GetAllDependencies(pkg.ImportPath, TypesConfig)
	if err != nil {
		return err
	}

	for _, dep := range dependencies {
		file.WriteString("Go$packages[\"" + dep.Path() + "\"] = (function() {\n")
		file.Write(Packages[dep.Path()].JavaScriptCode)
		file.WriteString("})();\n")
	}

	translator.WriteInterfaces(dependencies, file, false)

	for _, dep := range dependencies {
		file.WriteString("Go$packages[\"" + dep.Path() + "\"].init();\n")
	}
	file.WriteString("Go$packages[\"" + pkg.ImportPath + "\"].main();\n")

	return nil
}
