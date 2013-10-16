package main

import (
	"code.google.com/p/go.tools/go/exact"
	"code.google.com/p/go.tools/go/types"
	"fmt"
	"github.com/neelance/gopherjs/translator"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path"
)

func main() {
	b := &Builder{
		BuildContext: &build.Context{
			GOROOT:        build.Default.GOROOT,
			GOPATH:        build.Default.GOPATH,
			GOOS:          build.Default.GOOS,
			GOARCH:        build.Default.GOARCH,
			Compiler:      "gc",
			InstallSuffix: "js",
		},
		TypesConfig: &types.Config{
			Packages: make(map[string]*types.Package),
		},
		FileSet:  token.NewFileSet(),
		Packages: make(map[string]*BuilderPackage),
	}

	var pkg *BuilderPackage
	cmd := "help"
	if len(os.Args) >= 2 {
		cmd = os.Args[1]
	}
	switch cmd {
	case "install":
		buildPkg, err := b.BuildContext.Import(os.Args[2], "", 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		pkg = &BuilderPackage{Package: buildPkg}
		if pkg.IsCommand() {
			pkg.PkgObj = pkg.BinDir + "/" + path.Base(pkg.ImportPath) + ".js"
		}

	case "build", "run":
		filename := os.Args[2]
		file, err := parser.ParseFile(b.FileSet, filename, nil, parser.ImportsOnly)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		imports := make([]string, len(file.Imports))
		for i, imp := range file.Imports {
			imports[i] = imp.Path.Value[1 : len(imp.Path.Value)-1]
		}

		basename := path.Base(filename)
		pkg = &BuilderPackage{
			Package: &build.Package{
				Name:       "main",
				ImportPath: "main",
				Imports:    imports,
				Dir:        path.Dir(filename),
				GoFiles:    []string{basename},
				PkgObj:     basename[:len(basename)-3] + ".js",
			},
		}

	case "help":
		os.Stderr.WriteString(`GopherJS is a tool for compiling Go source code to JavaScript.

Usage:

    gopherjs command [arguments]

The commands are:

    build       compile packages and dependencies
    install     compile and install packages and dependencies
    run         compile and run Go program

`)
		return

	default:
		fmt.Fprintf(os.Stderr, "gopherjs: unknown subcommand \"%s\"\nRun 'gopherjs help' for usage.\n", cmd)
		return
	}

	err := b.BuildPackage(pkg)
	if err != nil {
		if err == PkgObjUpToDate {
			return
		}
		if list, isList := err.(translator.ErrorList); isList {
			for _, entry := range list {
				fmt.Fprintln(os.Stderr, entry)
			}
			return
		}
		fmt.Fprintln(os.Stderr, err)
		return
	}

	switch cmd {
	case "build", "install":
		if !pkg.IsCommand() {
			return // already stored by BuildPackage
		}

		webMode := false
		webModeConst := b.TypesConfig.Packages[pkg.ImportPath].Scope().Lookup("gopherjsWebMode")
		if webModeConst != nil {
			webMode = exact.BoolVal(webModeConst.(*types.Const).Val())
		}

		if err := os.MkdirAll(path.Dir(pkg.PkgObj), 0777); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		var perm os.FileMode = 0666
		if !webMode {
			perm = 0777
		}
		file, err := os.OpenFile(pkg.PkgObj, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		if !webMode {
			fmt.Fprintln(file, "#!/usr/bin/env node")
		}
		fmt.Fprintln(file, `"use strict";`)
		fmt.Fprintf(file, "var Go$webMode = %t;\n", webMode)
		if webMode {
			fmt.Fprintln(file, `var Go$syscall = function() { throw "Syscalls not available in browser." };`)
		}
		file.Write(pkg.JavaScriptCode)
		file.Close()
	case "run":
		node := exec.Command("node")
		pipe, _ := node.StdinPipe()
		node.Stdout = os.Stdout
		node.Stderr = os.Stderr
		err = node.Start()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		fmt.Fprintln(pipe, `"use strict";`)
		fmt.Fprintln(pipe, "var Go$webMode = false;")
		pipe.Write(pkg.JavaScriptCode)
		pipe.Close()
		node.Wait()
	}
}
