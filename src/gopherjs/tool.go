package main

import (
	"code.google.com/p/go.tools/go/types"
	"flag"
	"fmt"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"gopherjs/gcexporter"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"time"
)

func main() {
	var previousErr string
	var t *Translator
	t = &Translator{
		buildContext: &build.Context{
			GOROOT:        build.Default.GOROOT,
			GOPATH:        build.Default.GOPATH,
			GOOS:          build.Default.GOOS,
			GOARCH:        build.Default.GOARCH,
			Compiler:      "gc",
			InstallSuffix: "js",
			ReadDir:       ioutil.ReadDir,
			OpenFile:      func(name string) (io.ReadCloser, error) { return os.Open(name) },
		},
		typesConfig: &types.Config{
			Packages: make(map[string]*types.Package),
			Import: func(imports map[string]*types.Package, path string) (*types.Package, error) {
				return imports[path], nil
			},
			Error: func(err error) {
				if err.Error() != previousErr {
					fmt.Println(err.Error())
				}
				previousErr = err.Error()
			},
		},
		getModTime: func(name string) time.Time {
			if name == "" {
				name = os.Args[0] // gopherjs itself
			}
			fileInfo, err := os.Stat(os.Args[0])
			if err != nil {
				return time.Unix(0, 0)
			}
			return fileInfo.ModTime()
		},
		storePackage: func(pkg *GopherPackage) error {
			if err := os.MkdirAll(path.Dir(pkg.PkgObj), 0777); err != nil {
				return err
			}
			var perm os.FileMode = 0666
			if pkg.IsCommand() {
				perm = 0777
			}
			file, err := os.OpenFile(pkg.PkgObj, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
			if err != nil {
				return err
			}
			if pkg.IsCommand() {
				file.Write([]byte("#!/usr/bin/env node\n"))
			}
			file.Write(pkg.JavaScriptCode)
			file.WriteString("$$\n")
			if !pkg.IsCommand() {
				gcexporter.Write(t.typesConfig.Packages[pkg.ImportPath], file)
			}
			file.Close()
			return nil
		},
		fileSet:  token.NewFileSet(),
		packages: make(map[string]*GopherPackage),
	}

	flag.Parse()

	var pkg *GopherPackage
	cmd := flag.Arg(0)
	switch cmd {
	case "install":
		buildPkg, err := t.buildContext.Import(flag.Arg(1), "", 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		pkg = &GopherPackage{Package: buildPkg}
		pkg.PkgObj = pkg.BinDir + "/" + path.Base(pkg.ImportPath) + ".js"

	case "build", "run":
		filename := flag.Arg(1)
		file, err := parser.ParseFile(t.fileSet, filename, nil, parser.ImportsOnly)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		imports := make([]string, len(file.Imports))
		for i, imp := range file.Imports {
			imports[i] = imp.Path.Value[1 : len(imp.Path.Value)-1]
		}

		basename := path.Base(filename)
		pkgObj := ""
		if cmd == "build" {
			pkgObj = basename[:len(basename)-3] + ".js"
		}
		pkg = &GopherPackage{
			Package: &build.Package{
				Name:       "main",
				ImportPath: "main",
				Imports:    imports,
				Dir:        path.Dir(filename),
				GoFiles:    []string{basename},
				PkgObj:     pkgObj,
			},
		}

	case "help", "":
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

	err := t.buildPackage(pkg)
	if err != nil {
		list, isList := err.(scanner.ErrorList)
		if !isList {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		for _, entry := range list {
			fmt.Fprintln(os.Stderr, entry)
		}
		return
	}

	if cmd == "run" {
		node := exec.Command("node")
		pipe, _ := node.StdinPipe()
		node.Stdout = os.Stdout
		node.Stderr = os.Stderr
		err = node.Start()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		pipe.Write(pkg.JavaScriptCode)
		pipe.Close()
		node.Wait()
	}
}
