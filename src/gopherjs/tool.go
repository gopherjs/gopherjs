package main

import (
	"flag"
	"fmt"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path"
	"time"
)

type Translator struct {
	packages map[string]*GopherPackage
}

type GopherPackage struct {
	*build.Package
	importedPackages []*GopherPackage
	srcLastModified  time.Time
	archiveFile      string
}

func main() {
	var pkg *GopherPackage
	var out io.Writer

	fileSet := token.NewFileSet()

	flag.Parse()

	switch flag.Arg(0) {
	case "install":
		buildPkg, err := build.Import(flag.Arg(1), "", 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		pkg = &GopherPackage{Package: buildPkg}

	case "build", "run":
		filename := flag.Arg(1)
		file, err := parser.ParseFile(fileSet, filename, nil, parser.ImportsOnly)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		imports := make([]string, len(file.Imports))
		for i, imp := range file.Imports {
			imports[i] = imp.Path.Value[1 : len(imp.Path.Value)-1]
		}

		basename := path.Base(filename)
		pkg = &GopherPackage{
			Package: &build.Package{
				Name:    "main",
				Imports: imports,
				Dir:     path.Dir(filename),
				GoFiles: []string{basename},
			},
			archiveFile: basename[:len(basename)-3] + ".js",
		}

		if flag.Arg(0) == "run" {
			node := exec.Command("node")
			pipe, _ := node.StdinPipe()
			out = pipe
			node.Stdout = os.Stdout
			node.Stderr = os.Stderr
			err = node.Start()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			defer node.Wait()
			defer pipe.Close()
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
		fmt.Fprintf(os.Stderr, "gopherjs: unknown subcommand \"%s\"\nRun 'gopherjs help' for usage.\n", flag.Arg(0))
		return
	}

	t := &Translator{
		packages: make(map[string]*GopherPackage),
	}
	t.packages["reflect"] = &GopherPackage{Package: &build.Package{}}
	t.packages["go/doc"] = &GopherPackage{Package: &build.Package{}}

	err := t.buildPackage(pkg, fileSet, out)
	if err != nil {
		list, isList := err.(scanner.ErrorList)
		if !isList {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		for _, entry := range list {
			fmt.Fprintln(os.Stderr, entry)
		}
	}
}

func (t *Translator) buildPackage(pkg *GopherPackage, fileSet *token.FileSet, out io.Writer) error {
	fileInfo, err := os.Stat(os.Args[0]) // gopherjs itself
	if err != nil {
		return err
	}
	pkg.srcLastModified = fileInfo.ModTime()

	pkg.importedPackages = make([]*GopherPackage, len(pkg.Imports))
	for i, importedPkg := range pkg.Imports {
		if _, found := t.packages[importedPkg]; !found {
			otherPkg, err := build.Import(importedPkg, pkg.Dir, 0)
			if err != nil {
				return err
			}
			if err := t.buildPackage(&GopherPackage{Package: otherPkg}, fileSet, nil); err != nil {
				return err
			}
		}

		compiledPkg := t.packages[importedPkg]
		pkg.importedPackages[i] = compiledPkg
		if compiledPkg.srcLastModified.After(pkg.srcLastModified) {
			pkg.srcLastModified = compiledPkg.srcLastModified
		}
	}

	for _, name := range pkg.GoFiles {
		fileInfo, err := os.Stat(pkg.Dir + "/" + name)
		if err != nil {
			return err
		}
		if fileInfo.ModTime().After(pkg.srcLastModified) {
			pkg.srcLastModified = fileInfo.ModTime()
		}
	}

	if pkg.archiveFile == "" {
		pkg.archiveFile = pkg.PkgRoot + "/gopher_js/" + pkg.ImportPath + ".a"
		if pkg.IsCommand() {
			pkg.archiveFile = pkg.BinDir + "/" + path.Base(pkg.ImportPath) + ".js"
		}
	}

	t.packages[pkg.ImportPath] = pkg

	fileInfo, err = os.Stat(pkg.archiveFile)
	if err == nil {
		if fileInfo.ModTime().After(pkg.srcLastModified) {
			return nil
		}
	}

	return pkg.translate(fileSet, out)
}
