package main

import (
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

var buildContext = &build.Context{
	GOROOT:        build.Default.GOROOT,
	GOPATH:        build.Default.GOPATH,
	GOOS:          build.Default.GOOS,
	GOARCH:        build.Default.GOARCH,
	Compiler:      "gc",
	InstallSuffix: "js",
}
var typesConfig = &types.Config{
	Packages: make(map[string]*types.Package),
}
var fileSet = token.NewFileSet()
var packages = make(map[string]*Package)
var installMode = false

func main() {
	flag.Parse()

	cmd := flag.Arg(0)
	switch cmd {
	case "build":
		buildFlags := flag.NewFlagSet("build", flag.ContinueOnError)
		var pkgObj string
		buildFlags.StringVar(&pkgObj, "o", "", "")
		buildFlags.Parse(flag.Args()[1:])

		if buildFlags.NArg() == 0 {
			wd, err := os.Getwd()
			handleError(err)
			buildPkg, err := buildContext.ImportDir(wd, 0)
			handleError(err)
			pkg := &Package{Package: buildPkg}
			pkg.ImportPath = wd
			if pkgObj == "" {
				pkgObj = path.Base(wd) + ".js"
			}
			pkg.PkgObj = pkgObj
			err = buildPackage(pkg)
			handleError(err)
			os.Exit(0)
		}

		if strings.HasSuffix(buildFlags.Arg(0), ".go") {
			for _, arg := range buildFlags.Args() {
				if !strings.HasSuffix(arg, ".go") {
					fmt.Fprintln(os.Stderr, "named files must be .go files")
					os.Exit(1)
				}
			}
			if pkgObj == "" {
				basename := path.Base(buildFlags.Arg(0))
				pkgObj = basename[:len(basename)-3] + ".js"
			}
			err := buildFiles(buildFlags.Args(), pkgObj)
			handleError(err)
			os.Exit(0)
		}

		for _, pkgPath := range buildFlags.Args() {
			buildPkg, err := buildContext.Import(pkgPath, "", 0)
			handleError(err)
			pkg := &Package{Package: buildPkg}
			if pkgObj == "" {
				pkgObj = path.Base(buildFlags.Arg(0)) + ".js"
			}
			pkg.PkgObj = pkgObj
			err = buildPackage(pkg)
			handleError(err)
		}

	case "install":
		installFlags := flag.NewFlagSet("install", flag.ContinueOnError)
		installFlags.Parse(flag.Args()[1:])

		installMode = true
		for _, pkgPath := range installFlags.Args() {
			buildPkg, err := buildContext.Import(pkgPath, "", 0)
			handleError(err)
			pkg := &Package{Package: buildPkg}
			if pkg.IsCommand() {
				pkg.PkgObj = pkg.BinDir + "/" + path.Base(pkg.ImportPath) + ".js"
			}
			err = buildPackage(pkg)
			handleError(err)
		}
		os.Exit(0)

	case "run":
		lastSourceArg := 1
		for {
			if !strings.HasSuffix(flag.Arg(lastSourceArg), ".go") {
				break
			}
			lastSourceArg += 1
		}
		if lastSourceArg == 1 {
			fmt.Fprintln(os.Stderr, "gopherjs run: no go files listed")
			os.Exit(1)
		}

		tempfile, err := ioutil.TempFile("", path.Base(flag.Arg(1))+".")
		handleError(err)
		defer func() {
			tempfile.Close()
			os.Remove(tempfile.Name())
		}()

		err = buildFiles(flag.Args()[1:lastSourceArg], tempfile.Name())
		handleError(err)

		node := exec.Command("node", append([]string{tempfile.Name()}, flag.Args()[lastSourceArg:]...)...)
		node.Stdin = os.Stdin
		node.Stdout = os.Stdout
		node.Stderr = os.Stderr
		if err = node.Run(); err != nil {
			if e, isExitError := err.(*exec.ExitError); isExitError {
				os.Exit(e.Sys().(syscall.WaitStatus).ExitStatus())
			}
			handleError(err)
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
				err := buildFiles([]string{toolFlags.Arg(0)}, basename[:len(basename)-3]+".js")
				handleError(err)
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

func handleError(err error) {
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

func buildFiles(filenames []string, pkgObj string) error {
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
	return buildPackage(pkg)
}

func buildPackage(pkg *Package) error {
	if pkg.ImportPath == "unsafe" {
		typesConfig.Packages["unsafe"] = types.Unsafe
		return nil
	}

	typesConfig.Import = func(imports map[string]*types.Package, path string) (*types.Package, error) {
		if _, found := packages[path]; found {
			return imports[path], nil
		}

		otherPkg, err := buildContext.Import(path, pkg.Dir, build.AllowBinary)
		if err != nil {
			return nil, err
		}
		pkg := &Package{Package: otherPkg}
		packages[path] = pkg
		if err := buildPackage(pkg); err != nil {
			return nil, err
		}

		return imports[path], nil
	}

	if installMode {
		if fileInfo, err := os.Stat(os.Args[0]); err == nil { // gopherjs itself
			pkg.SrcModTime = fileInfo.ModTime()
		}

		for _, importedPkgPath := range pkg.Imports {
			_, err := typesConfig.Import(typesConfig.Packages, importedPkgPath)
			if err != nil {
				return err
			}
			impModeTime := packages[importedPkgPath].SrcModTime
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

			pkg.JavaScriptCode, _, err = translator.ReadArchive(typesConfig.Packages, pkg.PkgObj, pkg.ImportPath, objFile)
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
	if errList != nil {
		return errList
	}

	var err error
	pkg.JavaScriptCode, err = translator.TranslatePackage(pkg.ImportPath, files, fileSet, typesConfig)
	if err != nil {
		return err
	}

	if !pkg.IsCommand() {
		if installMode {
			if err := os.MkdirAll(path.Dir(pkg.PkgObj), 0777); err != nil {
				return err
			}
			file, err := os.Create(pkg.PkgObj)
			if err != nil {
				return err
			}
			defer file.Close()
			translator.WriteArchive(pkg.JavaScriptCode, typesConfig.Packages[pkg.ImportPath], file)
		}
		return nil
	}

	if err := os.MkdirAll(path.Dir(pkg.PkgObj), 0777); err != nil {
		return err
	}
	file, err := os.Create(pkg.PkgObj)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Fprintln(file, `"use strict";`)
	file.WriteString(strings.TrimSpace(translator.Prelude))
	file.WriteString("\n")

	packages[pkg.ImportPath] = pkg
	dependencies, err := translator.GetAllDependencies(pkg.ImportPath, typesConfig)
	if err != nil {
		return err
	}

	for _, dep := range dependencies {
		file.WriteString("Go$packages[\"" + dep.Path() + "\"] = (function() {\n")
		file.Write(packages[dep.Path()].JavaScriptCode)
		file.WriteString("})();\n")
	}

	translator.WriteInterfaces(dependencies, file, false)

	for _, dep := range dependencies {
		file.WriteString("Go$packages[\"" + dep.Path() + "\"].init();\n")
	}
	file.WriteString("Go$packages[\"" + pkg.ImportPath + "\"].main();\n")

	return nil
}
