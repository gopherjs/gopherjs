package main

import (
	"bitbucket.org/kardianos/osext"
	"flag"
	"fmt"
	"github.com/gopherjs/gopherjs/translator"
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

type packageData struct {
	*build.Package
	SrcModTime time.Time
	UpToDate   bool
	Archive    *translator.Archive
}

type importCError struct{}

func (e *importCError) Error() string {
	return `importing "C" is not supported by GopherJS`
}

var currentDirectory, goRoot, goPath string

func init() {
	var err error
	currentDirectory, err = os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	currentDirectory, err = filepath.EvalSymlinks(currentDirectory)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func buildImport(path string, mode build.ImportMode) (*build.Package, error) {
	if path == "C" {
		return nil, &importCError{}
	}

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
	if pkg.IsCommand() {
		pkg.PkgObj = filepath.Join(pkg.BinDir, filepath.Base(pkg.ImportPath)+".js")
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
var packages = make(map[string]*packageData)
var installMode = false
var verboseInstall = false
var packagesToTest = make(map[string]bool)

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
			buildContext := &build.Context{
				GOROOT:   build.Default.GOROOT,
				GOPATH:   build.Default.GOPATH,
				GOOS:     build.Default.GOOS,
				GOARCH:   "js",
				Compiler: "gc",
			}
			buildPkg, err := buildContext.ImportDir(currentDirectory, 0)
			if err != nil {
				return err
			}
			pkg := &packageData{Package: buildPkg}
			pkg.ImportPath = currentDirectory
			if err := buildPackage(pkg); err != nil {
				return err
			}
			if pkgObj == "" {
				pkgObj = filepath.Base(currentDirectory) + ".js"
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
			buildPkg, err := buildImport(filepath.ToSlash(pkgPath), 0)
			if err != nil {
				return err
			}
			pkg := &packageData{Package: buildPkg}
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
		installFlags.BoolVar(&verboseInstall, "v", false, "verbose")
		all := installFlags.Bool("all", false, "install all packages in GOROOT")
		installFlags.Parse(flag.Args()[1:])

		installMode = true
		pkgs := installFlags.Args()
		if *all {
			dir := filepath.Join(build.Default.GOROOT, "src", "pkg")
			err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() && path != dir {
					pkgPath := path[len(dir)+1:]
					if info.Name()[0] == '.' || info.Name() == "testdata" || pkgPath == "builtin" {
						return filepath.SkipDir
					}
					pkgs = append(pkgs, pkgPath)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}

		if len(pkgs) == 0 {
			srcDir, err := filepath.EvalSymlinks(filepath.Join(build.Default.GOPATH, "src"))
			if err != nil {
				return err
			}
			if !strings.HasPrefix(currentDirectory, srcDir) {
				return fmt.Errorf("gopherjs install: no install location for directory %s outside GOPATH", currentDirectory)
			}
			pkgPath, err := filepath.Rel(srcDir, currentDirectory)
			if err != nil {
				return err
			}
			pkgs = []string{pkgPath}
		}
		for _, pkgPath := range pkgs {
			if _, err := importPackage(filepath.ToSlash(pkgPath)); err != nil {
				switch err.(type) {
				case *build.NoGoError, *importCError:
					if *all {
						continue
					}
				}
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
			lastSourceArg++
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
		verbose := testFlags.Bool("v", false, "verbose")
		short := testFlags.Bool("short", false, "short")
		testFlags.Parse(flag.Args()[1:])

		for _, pkgPath := range testFlags.Args() {
			packagesToTest[filepath.ToSlash(pkgPath)] = true
		}

		mainPkg := &packageData{
			Package: &build.Package{
				Name:       "main",
				ImportPath: "main",
			},
			Archive: &translator.Archive{
				ImportPath: "main",
			},
		}
		packages["main"] = mainPkg
		translator.NewEmptyTypesPackage("main")
		testingOutput, _ := importPackage("testing")
		mainPkg.Archive.AddDependenciesOf(testingOutput)

		mainFunc := []byte("go$pkg.main = function() {\ngo$packages[\"flag\"].Parse();\n")
		for _, pkgPath := range testFlags.Args() {
			pkgPath = filepath.ToSlash(pkgPath)

			var names []string
			var tests []string
			collectTests := func(pkg *packageData) {
				for _, f := range pkg.Archive.Functions {
					if strings.HasPrefix(f.Name, "Test") {
						names = append(names, f.Name)
						tests = append(tests, fmt.Sprintf(`go$packages["%s"].%s`, pkg.ImportPath, f.Name))
					}
				}
				mainPkg.Archive.AddDependenciesOf(pkg.Archive)
			}

			if _, err := importPackage(pkgPath); err != nil {
				return err
			}
			pkg := packages[pkgPath]
			collectTests(pkg)

			if len(pkg.XTestGoFiles) != 0 {
				testPkg := &packageData{Package: &build.Package{
					ImportPath: pkg.ImportPath + "_test",
					Dir:        pkg.Dir,
					GoFiles:    pkg.XTestGoFiles,
				}}
				if err := buildPackage(testPkg); err != nil {
					return err
				}
				collectTests(testPkg)
			}

			mainFunc = append(mainFunc, []byte(fmt.Sprintf(`go$packages["testing"].RunTests2("%s", "%s", ["%s"], [%s]);`+"\n", pkg.ImportPath, pkg.Dir, strings.Join(names, `", "`), strings.Join(tests, ", ")))...)
		}
		mainFunc = append(mainFunc, []byte("};\n")...)
		mainPkg.Archive.Functions = []translator.Function{
			translator.Function{Name: "main", Code: mainFunc},
			translator.Function{Name: "init", Code: []byte("go$pkg.init = function() {};\n")},
		}
		mainPkg.Archive.AddDependency("main")

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
		if *verbose {
			args = append(args, "-test.v")
		}
		if *short {
			args = append(args, "-test.short")
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
		fmt.Fprintf(os.Stderr, "gopherjs: unknown subcommand \"%s\"\nRun 'gopherjs help' for usage.\n", cmd)
		return nil

	}
}

func buildFiles(filenames []string, pkgObj string) error {
	pkg := &packageData{
		Package: &build.Package{
			Name:       "main",
			ImportPath: "main",
			Dir:        currentDirectory,
			GoFiles:    filenames,
		},
	}

	if err := buildPackage(pkg); err != nil {
		return err
	}
	return writeCommandPackage(pkg, pkgObj)
}

func importPackage(path string) (*translator.Archive, error) {
	if pkg, found := packages[path]; found {
		return pkg.Archive, nil
	}

	otherPkg, err := buildImport(path, build.AllowBinary)
	if err != nil {
		return nil, err
	}
	pkg := &packageData{Package: otherPkg}
	if err := buildPackage(pkg); err != nil {
		return nil, err
	}

	return pkg.Archive, nil
}

func buildPackage(pkg *packageData) error {
	if pkg.ImportPath == "unsafe" {
		return nil
	}

	if pkg.PkgObj != "" && !packagesToTest[pkg.ImportPath] {
		var fileInfo os.FileInfo
		gopherjsBinary, err := osext.Executable()
		if err == nil {
			fileInfo, err = os.Stat(gopherjsBinary)
			if err == nil {
				pkg.SrcModTime = fileInfo.ModTime()
			}
		}
		if err != nil {
			os.Stderr.WriteString("Could not get GopherJS binary's modification timestamp. Please report issue.\n")
			pkg.SrcModTime = time.Now()
		}

		for _, importedPkgPath := range pkg.Imports {
			if importedPkgPath == "unsafe" {
				continue
			}
			_, err := importPackage(importedPkgPath)
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

			pkg.Archive, err = translator.ReadArchive(pkg.PkgObj, pkg.ImportPath, objFile)
			if err != nil {
				return err
			}
			packages[pkg.ImportPath] = pkg

			return nil
		}
	}

	if verboseInstall {
		fmt.Println(pkg.ImportPath)
	}

	var files []*ast.File
	var errList translator.ErrorList
	names := pkg.GoFiles
	if packagesToTest[pkg.ImportPath] {
		names = append(names, pkg.TestGoFiles...)
	}
	for _, name := range names {
		if pkg.ImportPath == "runtime" && strings.HasPrefix(name, "zgoarch_") {
			file, _ := parser.ParseFile(fileSet, name, "package runtime\nconst theGoarch = `js`\n", 0)
			files = append(files, file)
			continue
		}
		if pkg.ImportPath == "crypto/rc4" && name == "rc4_ref.go" { // apply patch https://codereview.appspot.com/40540049/
			file, _ := parser.ParseFile(fileSet, name, "package rc4\nfunc (c *Cipher) XORKeyStream(dst, src []byte) {\ni, j := c.i, c.j\nfor k, v := range src {\ni += 1\nj += uint8(c.s[i])\nc.s[i], c.s[j] = c.s[j], c.s[i]\ndst[k] = v ^ uint8(c.s[uint8(c.s[i]+c.s[j])])\n}\nc.i, c.j = i, j\n}\n", 0)
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
		if relname, err := filepath.Rel(currentDirectory, name); err == nil {
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
	pkg.Archive, err = translator.TranslatePackage(pkg.ImportPath, files, fileSet, importPackage)
	if err != nil {
		return err
	}
	packages[pkg.ImportPath] = pkg

	if installMode {
		if pkg.IsCommand() {
			return writeCommandPackage(pkg, pkg.PkgObj)
		}

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
		return nil
	}

	if pkg.ImportPath == "runtime" {
		fmt.Println(`note: run "gopherjs install -all -v" once to speed up builds`)
	}

	return nil
}

func writeLibraryPackage(pkg *packageData, pkgObj string) error {
	if err := os.MkdirAll(filepath.Dir(pkgObj), 0777); err != nil {
		return err
	}

	data, err := translator.WriteArchive(pkg.Archive)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(pkgObj, data, 0666)
}

func writeCommandPackage(pkg *packageData, pkgObj string) error {
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

	for _, depPath := range pkg.Archive.Dependencies {
		dep, err := importPackage(depPath)
		if err != nil {
			return err
		}
		dep.WriteCode(file)
	}

	translator.WriteInterfaces(pkg.Archive.Dependencies, file, false)

	for _, depPath := range pkg.Archive.Dependencies {
		file.WriteString("go$packages[\"" + depPath + "\"].init();\n")
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
