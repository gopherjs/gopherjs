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
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type Package struct {
	*build.Package
	SrcModTime     time.Time
	UpToDate       bool
	JavaScriptCode []byte
}

var typesConfig = &types.Config{
	Packages: make(map[string]*types.Package),
}

func init() {
	typesConfig.Import = func(imports map[string]*types.Package, path string) (*types.Package, error) {
		if _, found := packages[path]; found {
			return imports[path], nil
		}

		otherPkg, err := buildImport(path, build.AllowBinary)
		if err != nil {
			return nil, err
		}
		pkg := &Package{Package: otherPkg}
		if err := buildPackage(pkg); err != nil {
			return nil, err
		}

		return imports[path], nil
	}
}

func buildImport(path string, mode build.ImportMode) (*build.Package, error) {
	buildContext := &build.Context{
		GOROOT:   build.Default.GOROOT,
		GOPATH:   build.Default.GOPATH,
		GOOS:     build.Default.GOOS,
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

var fileSet = token.NewFileSet()
var packages = make(map[string]*Package)
var installMode = false

func main() {
	switch err := tool().(type) {
	case nil:
		os.Exit(0)
	case translator.ErrorList:
		for _, entry := range err {
			fmt.Fprintln(os.Stderr, entry)
		}
		os.Exit(1)
	case *exec.ExitError:
		os.Exit(err.Sys().(syscall.WaitStatus).ExitStatus())
	default:
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func tool() error {
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
			if err != nil {
				return err
			}
			buildContext := &build.Context{
				GOROOT:   build.Default.GOROOT,
				GOPATH:   build.Default.GOPATH,
				GOOS:     build.Default.GOOS,
				GOARCH:   "js",
				Compiler: "gc",
			}
			buildPkg, err := buildContext.ImportDir(wd, 0)
			if err != nil {
				return err
			}
			pkg := &Package{Package: buildPkg}
			pkg.ImportPath = wd
			if err := buildPackage(pkg); err != nil {
				return err
			}
			if pkgObj == "" {
				pkgObj = filepath.Base(wd) + ".js"
			}
			if err := writeCommandPackage(pkg, pkgObj); err != nil {
				return err
			}
			return nil
		}

		if strings.HasSuffix(buildFlags.Arg(0), ".go") {
			for _, arg := range buildFlags.Args() {
				if !strings.HasSuffix(arg, ".go") {
					return fmt.Errorf("named files must be .go files")
				}
			}
			if pkgObj == "" {
				basename := filepath.Base(buildFlags.Arg(0))
				pkgObj = basename[:len(basename)-3] + ".js"
			}
			if err := buildFiles(buildFlags.Args(), pkgObj); err != nil {
				return err
			}
			return nil
		}

		for _, pkgPath := range buildFlags.Args() {
			buildPkg, err := buildImport(pkgPath, 0)
			if err != nil {
				return err
			}
			pkg := &Package{Package: buildPkg}
			if err := buildPackage(pkg); err != nil {
				return err
			}
			if pkgObj == "" {
				pkgObj = filepath.Base(buildFlags.Arg(0)) + ".js"
			}
			if err := writeCommandPackage(pkg, pkgObj); err != nil {
				return err
			}
		}
		return nil

	case "install":
		installFlags := flag.NewFlagSet("install", flag.ContinueOnError)
		installFlags.Parse(flag.Args()[1:])

		installMode = true
		for _, pkgPath := range installFlags.Args() {
			buildPkg, err := buildImport(pkgPath, 0)
			if err != nil {
				return err
			}
			pkg := &Package{Package: buildPkg}
			if pkg.IsCommand() {
				pkg.PkgObj = filepath.Join(pkg.BinDir, filepath.Base(pkg.ImportPath)+".js")
			}
			if err := buildPackage(pkg); err != nil {
				return err
			}
			if err := writeCommandPackage(pkg, pkg.PkgObj); err != nil {
				return err
			}
		}
		return nil

	case "run":
		lastSourceArg := 1
		for {
			if !strings.HasSuffix(flag.Arg(lastSourceArg), ".go") {
				break
			}
			lastSourceArg += 1
		}
		if lastSourceArg == 1 {
			return fmt.Errorf("gopherjs run: no go files listed")
		}

		tempfile, err := ioutil.TempFile("", filepath.Base(flag.Arg(1))+".")
		if err != nil {
			return err
		}
		defer func() {
			tempfile.Close()
			os.Remove(tempfile.Name())
		}()

		if err := buildFiles(flag.Args()[1:lastSourceArg], tempfile.Name()); err != nil {
			return err
		}
		if err := runNode(tempfile.Name(), flag.Args()[lastSourceArg:], ""); err != nil {
			return err
		}
		return nil

	case "test":
		testFlags := flag.NewFlagSet("test", flag.ContinueOnError)
		short := testFlags.Bool("short", false, "")
		verbose := testFlags.Bool("v", false, "")
		testFlags.Parse(flag.Args()[1:])

		mainPkg := &Package{Package: &build.Package{
			Name:       "main",
			ImportPath: "main",
		}}
		packages["main"] = mainPkg
		mainPkgTypes := types.NewPackage("main", "main", types.NewScope(nil))
		testingPkgTypes, _ := typesConfig.Import(typesConfig.Packages, "testing")
		mainPkgTypes.SetImports([]*types.Package{testingPkgTypes})
		typesConfig.Packages["main"] = mainPkgTypes
		mainPkg.JavaScriptCode = []byte("go$pkg.main = function() {\ngo$packages[\"flag\"].Parse();\n")

		for _, pkgPath := range testFlags.Args() {
			buildPkg, err := buildImport(pkgPath, 0)
			if err != nil {
				return err
			}

			pkg := &Package{Package: buildPkg}
			pkg.GoFiles = append(pkg.GoFiles, pkg.TestGoFiles...)
			pkg.PkgObj = "" // do not load from disk
			if err := buildPackage(pkg); err != nil {
				return err
			}

			testPkg := &Package{Package: &build.Package{
				ImportPath: pkg.ImportPath + "_test",
				Dir:        pkg.Dir,
				GoFiles:    pkg.XTestGoFiles,
			}}
			if err := buildPackage(testPkg); err != nil {
				return err
			}

			var names []string
			var tests []string
			imports := mainPkgTypes.Imports()
			collectTests := func(pkg *Package) {
				pkgTypes := typesConfig.Packages[pkg.ImportPath]
				for _, name := range pkgTypes.Scope().Names() {
					_, isFunction := pkgTypes.Scope().Lookup(name).Type().(*types.Signature)
					if isFunction && strings.HasPrefix(name, "Test") {
						names = append(names, name)
						tests = append(tests, fmt.Sprintf(`go$packages["%s"].%s`, pkg.ImportPath, name))
					}
				}
				imports = append(imports, pkgTypes)
			}
			collectTests(pkg)
			collectTests(testPkg)
			mainPkg.JavaScriptCode = append(mainPkg.JavaScriptCode, []byte(fmt.Sprintf(`go$packages["testing"].RunTests2("%s", "%s", ["%s"], [%s]);`+"\n", pkg.ImportPath, pkg.Dir, strings.Join(names, `", "`), strings.Join(tests, ", ")))...)
			mainPkgTypes.SetImports(imports)
		}
		mainPkg.JavaScriptCode = append(mainPkg.JavaScriptCode, []byte("}; go$pkg.init = function() {};")...)

		tempfile, err := ioutil.TempFile("", "test.")
		if err != nil {
			return err
		}
		defer func() {
			tempfile.Close()
			os.Remove(tempfile.Name())
		}()

		if err := writeCommandPackage(mainPkg, tempfile.Name()); err != nil {
			return err
		}

		var args []string
		if *short {
			args = append(args, "-test.short")
		}
		if *verbose {
			args = append(args, "-test.v")
		}
		return runNode(tempfile.Name(), args, "")

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
				basename := filepath.Base(toolFlags.Arg(0))
				if err := buildFiles([]string{toolFlags.Arg(0)}, basename[:len(basename)-3]+".js"); err != nil {
					return err
				}
				return nil
			}
		}
		return fmt.Errorf("Tool not supported: " + tool)

	case "help", "":
		os.Stderr.WriteString(`GopherJS is a tool for compiling Go source code to JavaScript.

Usage:

    gopherjs command [arguments]

The commands are:

    build       compile packages and dependencies
    install     compile and install packages and dependencies
    run         compile and run Go program

`)
		return nil

	default:
		return fmt.Errorf("gopherjs: unknown subcommand \"%s\"\nRun 'gopherjs help' for usage.", cmd)

	}
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
		},
	}

	if err := buildPackage(pkg); err != nil {
		return err
	}
	return writeCommandPackage(pkg, pkgObj)
}

func buildPackage(pkg *Package) error {
	packages[pkg.ImportPath] = pkg
	if pkg.ImportPath == "unsafe" {
		typesConfig.Packages["unsafe"] = types.Unsafe
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

			objFile, err := ioutil.ReadFile(pkg.PkgObj)
			if err != nil {
				return err
			}

			pkg.JavaScriptCode, _, err = translator.ReadArchive(typesConfig.Packages, pkg.PkgObj, pkg.ImportPath, objFile)
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
		if pkg.ImportPath == "runtime" && strings.HasPrefix(name, "zgoarch_") {
			file, _ := parser.ParseFile(fileSet, name, "package runtime\nconst theGoarch = `js`\n", 0)
			files = append(files, file)
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
	if errList != nil {
		return errList
	}

	var err error
	pkg.JavaScriptCode, err = translator.TranslatePackage(pkg.ImportPath, files, fileSet, typesConfig)
	if err != nil {
		return err
	}

	if installMode && !pkg.IsCommand() {
		if err := writeLibraryPackage(pkg, pkg.PkgObj); err != nil {
			if strings.HasPrefix(pkg.PkgObj, build.Default.GOROOT) {
				// fall back to GOPATH
				if err := writeLibraryPackage(pkg, build.Default.GOPATH+pkg.PkgObj[len(build.Default.GOROOT):]); err != nil {
					return err
				}
				return nil
			}
			return err
		}
	}

	return nil
}

func writeLibraryPackage(pkg *Package, pkgObj string) error {
	if err := os.MkdirAll(filepath.Dir(pkgObj), 0777); err != nil {
		return err
	}

	data, err := translator.WriteArchive(pkg.JavaScriptCode, typesConfig.Packages[pkg.ImportPath])
	if err != nil {
		return err
	}

	return ioutil.WriteFile(pkgObj, data, 0666)
}

func writeCommandPackage(pkg *Package, pkgObj string) error {
	if !pkg.IsCommand() || pkg.UpToDate {
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

	dependencies, err := translator.GetAllDependencies(pkg.ImportPath, typesConfig)
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

func runNode(script string, args []string, dir string) error {
	node := exec.Command("node", append([]string{script}, args...)...)
	node.Dir = dir
	node.Stdin = os.Stdin
	node.Stdout = os.Stdout
	node.Stderr = os.Stderr
	return node.Run()
}
