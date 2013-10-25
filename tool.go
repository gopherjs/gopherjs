package main

import (
	"code.google.com/p/go.tools/go/exact"
	"code.google.com/p/go.tools/go/types"
	"flag"
	"fmt"
	"github.com/neelance/gopherjs/translator"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path"
)

type Mode int

const (
	Build Mode = iota
	Run
	Install
)

func main() {
	flag.Parse()

	cmd := flag.Arg(0)
	switch cmd {
	case "build":
		err := Do(Build, flag.Arg(1))
		HandleError(err)
		os.Exit(0)

	case "install":
		err := Do(Install, flag.Arg(1))
		HandleError(err)
		os.Exit(0)

	case "run":
		err := Do(Run, flag.Arg(1))
		HandleError(err)
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
				err := Do(Build, toolFlags.Arg(0))
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

func Do(mode Mode, filename string) error {
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
	switch mode {
	case Install:
		buildPkg, err := b.BuildContext.Import(filename, "", 0)
		if err != nil {
			return err
		}
		pkg = &BuilderPackage{Package: buildPkg}
		if pkg.IsCommand() {
			pkg.PkgObj = pkg.BinDir + "/" + path.Base(pkg.ImportPath) + ".js"
		}

	case Build, Run:
		file, err := parser.ParseFile(b.FileSet, filename, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}

		imports := make([]string, len(file.Imports))
		for i, imp := range file.Imports {
			imports[i] = imp.Path.Value[1 : len(imp.Path.Value)-1]
		}

		basename := path.Base(filename)
		pkgObj := ""
		if mode == Build {
			pkgObj = basename[:len(basename)-3] + ".js"
		}
		pkg = &BuilderPackage{
			Package: &build.Package{
				Name:       "main",
				ImportPath: "main",
				Imports:    imports,
				Dir:        path.Dir(filename),
				GoFiles:    []string{basename},
				PkgObj:     pkgObj,
			},
		}
	}

	err := b.BuildPackage(pkg)
	if err != nil {
		if err == PkgObjUpToDate {
			return nil
		}
		return err
	}

	switch mode {
	case Build, Install:
		if !pkg.IsCommand() {
			return nil // already stored by BuildPackage
		}

		webMode := false
		webModeConst := b.TypesConfig.Packages[pkg.ImportPath].Scope().Lookup("gopherjsWebMode")
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
		if !webMode {
			fmt.Fprintln(file, "#!/usr/bin/env node")
		}
		fmt.Fprintln(file, `"use strict";`)
		fmt.Fprintf(file, "var Go$webMode = %t;\n", webMode)
		file.Write(pkg.JavaScriptCode)
		file.Close()

	case Run:
		node := exec.Command("node")
		pipe, _ := node.StdinPipe()
		node.Stdout = os.Stdout
		node.Stderr = os.Stderr
		err = node.Start()
		if err != nil {
			return err
		}
		fmt.Fprintln(pipe, `"use strict";`)
		fmt.Fprintln(pipe, "var Go$webMode = false;")
		pipe.Write(pkg.JavaScriptCode)
		pipe.Close()
		node.Wait()
	}

	return nil
}
