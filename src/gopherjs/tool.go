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
	packages map[string]*CompiledPackage
}

type CompiledPackage struct {
	*build.Package
	importedPackages []*CompiledPackage
	srcLastModified  time.Time
	archiveFile      string
}

func main() {
	var pkg *build.Package
	var out io.Writer

	fileSet := token.NewFileSet()

	flag.Parse()

	switch flag.Arg(0) {
	case "install":
		var err error
		pkg, err = build.Import(flag.Arg(1), "", 0)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
	case "run":
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

		pkg = &build.Package{
			Name:    "main",
			Imports: imports,
			Dir:     path.Dir(filename),
			GoFiles: []string{path.Base(filename)},
		}
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

	case "help", "":
		os.Stderr.WriteString(`GopherJS is a tool for compiling Go source code to JavaScript.

Usage:

    gopherjs command [arguments]

The commands are:

    install     compile and install packages and dependencies
    run         compile and run Go program

`)
		return

	default:
		fmt.Fprintf(os.Stderr, "gopherjs: unknown subcommand \"%s\"\nRun 'gopherjs help' for usage.\n", flag.Arg(0))
		return
	}

	t := &Translator{
		packages: make(map[string]*CompiledPackage),
	}
	t.packages["reflect"] = &CompiledPackage{Package: &build.Package{}}
	t.packages["go/doc"] = &CompiledPackage{Package: &build.Package{}}

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

func (t *Translator) buildPackage(pkg *build.Package, fileSet *token.FileSet, out io.Writer) error {
	fileInfo, err := os.Stat(os.Args[0]) // gopherjs itself
	if err != nil {
		return err
	}
	srcLastModified := fileInfo.ModTime()

	importedPackages := make([]*CompiledPackage, len(pkg.Imports))
	for i, importedPkg := range pkg.Imports {
		if _, found := t.packages[importedPkg]; !found {
			otherPkg, err := build.Import(importedPkg, pkg.Dir, 0)
			if err != nil {
				return err
			}
			if err := t.buildPackage(otherPkg, fileSet, nil); err != nil {
				return err
			}
		}

		compiledPkg := t.packages[importedPkg]
		importedPackages[i] = compiledPkg
		if compiledPkg.srcLastModified.After(srcLastModified) {
			srcLastModified = compiledPkg.srcLastModified
		}
	}

	for _, name := range pkg.GoFiles {
		fileInfo, err := os.Stat(pkg.Dir + "/" + name)
		if err != nil {
			return err
		}
		if fileInfo.ModTime().After(srcLastModified) {
			srcLastModified = fileInfo.ModTime()
		}
	}

	archiveFile := pkg.PkgRoot + "/gopher_js/" + pkg.ImportPath + ".a"
	if pkg.IsCommand() {
		archiveFile = pkg.BinDir + "/" + path.Base(pkg.ImportPath) + ".js"
	}

	p := &CompiledPackage{
		Package:          pkg,
		importedPackages: importedPackages,
		srcLastModified:  srcLastModified,
		archiveFile:      archiveFile,
	}
	t.packages[pkg.ImportPath] = p

	fileInfo, err = os.Stat(archiveFile)
	if err == nil {
		if fileInfo.ModTime().After(srcLastModified) {
			return nil
		}
	}

	return p.translate(fileSet, out)
}
