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
	"gopherjs/translator"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"time"
)

func main() {
	webMode := false
	flag.BoolVar(&webMode, "w", false, "")
	flag.Parse()

	var previousErr string
	var t *translator.Translator
	t = &translator.Translator{
		BuildContext: &build.Context{
			GOROOT:        build.Default.GOROOT,
			GOPATH:        build.Default.GOPATH,
			GOOS:          build.Default.GOOS,
			GOARCH:        build.Default.GOARCH,
			Compiler:      "gc",
			InstallSuffix: "js",
			ReadDir:       ioutil.ReadDir,
			OpenFile:      func(name string) (io.ReadCloser, error) { return os.Open(name) },
		},
		TypesConfig: &types.Config{
			Packages: make(map[string]*types.Package),
			Error: func(err error) {
				if err.Error() != previousErr {
					fmt.Println(err.Error())
				}
				previousErr = err.Error()
			},
		},
		GetModTime: func(name string) time.Time {
			if name == "" {
				name = os.Args[0] // gopherjs itself
			}
			fileInfo, err := os.Stat(name)
			if err != nil {
				return time.Unix(0, 0)
			}
			return fileInfo.ModTime()
		},
		StoreArchive: func(pkg *translator.GopherPackage) error {
			if err := os.MkdirAll(path.Dir(pkg.PkgObj), 0777); err != nil {
				return err
			}
			file, err := os.Create(pkg.PkgObj)
			if err != nil {
				return err
			}
			file.Write(pkg.JavaScriptCode)
			file.WriteString("$$\n")
			gcexporter.Write(t.TypesConfig.Packages[pkg.ImportPath], file)
			file.Close()
			return nil
		},
		FileSet:  token.NewFileSet(),
		Packages: make(map[string]*translator.GopherPackage),
	}

	var pkg *translator.GopherPackage
	cmd := flag.Arg(0)
	switch cmd {
	case "install":
		buildPkg, err := t.BuildContext.Import(flag.Arg(1), "", 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		pkg = &translator.GopherPackage{Package: buildPkg}
		if pkg.IsCommand() {
			pkg.PkgObj = pkg.BinDir + "/" + path.Base(pkg.ImportPath) + ".js"
		}

	case "build", "run":
		filename := flag.Arg(1)
		file, err := parser.ParseFile(t.FileSet, filename, nil, parser.ImportsOnly)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		imports := make([]string, len(file.Imports))
		for i, imp := range file.Imports {
			imports[i] = imp.Path.Value[1 : len(imp.Path.Value)-1]
		}

		basename := path.Base(filename)
		pkg = &translator.GopherPackage{
			Package: &build.Package{
				Name:       "main",
				ImportPath: "main",
				Imports:    imports,
				Dir:        path.Dir(filename),
				GoFiles:    []string{basename},
				PkgObj:     basename[:len(basename)-3] + ".js",
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

	err := t.BuildPackage(pkg)
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

	switch cmd {
	case "build", "install":
		if !pkg.IsCommand() {
			return // already stored by BuildPackage
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
			file.Write([]byte("#!/usr/bin/env node\n"))
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
		pipe.Write(pkg.JavaScriptCode)
		pipe.Close()
		node.Wait()
	}
}
