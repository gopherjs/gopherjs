package main

import (
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"code.google.com/p/go.tools/go/types"
	"github.com/neelance/gopherjs/api"
	"github.com/neelance/gopherjs/translator"
)

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
				GOOS:     "darwin",
				GOARCH:   "js",
				Compiler: "gc",
			}
			buildPkg, err := buildContext.ImportDir(wd, 0)
			if err != nil {
				return err
			}
			pkg := &api.Package{Package: buildPkg}
			pkg.ImportPath = wd
			if err := api.BuildPackage(pkg); err != nil {
				return err
			}
			if pkgObj == "" {
				pkgObj = filepath.Base(wd) + ".js"
			}
			if err := api.WriteCommandPackage(pkg, pkgObj); err != nil {
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
			if err := api.BuildFiles(buildFlags.Args(), pkgObj); err != nil {
				return err
			}
			return nil
		}

		for _, pkgPath := range buildFlags.Args() {
			buildPkg, err := api.BuildImport(pkgPath, 0)
			if err != nil {
				return err
			}
			pkg := &api.Package{Package: buildPkg}
			if err := api.BuildPackage(pkg); err != nil {
				return err
			}
			if pkgObj == "" {
				pkgObj = filepath.Base(buildFlags.Arg(0)) + ".js"
			}
			if err := api.WriteCommandPackage(pkg, pkgObj); err != nil {
				return err
			}
		}
		return nil

	case "install":
		installFlags := flag.NewFlagSet("install", flag.ContinueOnError)
		installFlags.Parse(flag.Args()[1:])

		api.InstallMode = true
		for _, pkgPath := range installFlags.Args() {
			buildPkg, err := api.BuildImport(pkgPath, 0)
			if err != nil {
				return err
			}
			pkg := &api.Package{Package: buildPkg}
			if pkg.IsCommand() {
				pkg.PkgObj = filepath.Join(pkg.BinDir, filepath.Base(pkg.ImportPath)+".js")
			}
			if err := api.BuildPackage(pkg); err != nil {
				return err
			}
			if err := api.WriteCommandPackage(pkg, pkg.PkgObj); err != nil {
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

		if err := api.BuildFiles(flag.Args()[1:lastSourceArg], tempfile.Name()); err != nil {
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

		mainPkg := &api.Package{Package: &build.Package{
			ImportPath: "main",
		}}
		api.SetPackage("main", mainPkg)
		mainPkgTypes := types.NewPackage("main", "main", types.NewScope(nil))
		testingPkgTypes, _ := api.TypesConfig.Import(api.TypesConfig.Packages, "testing")
		mainPkgTypes.SetImports([]*types.Package{testingPkgTypes})
		api.TypesConfig.Packages["main"] = mainPkgTypes
		mainPkg.JavaScriptCode = []byte("go$pkg.main = function() {\ngo$packages[\"flag\"].Parse();\n")

		for _, pkgPath := range testFlags.Args() {
			buildPkg, err := api.BuildImport(pkgPath, 0)
			if err != nil {
				return err
			}

			pkg := &api.Package{Package: buildPkg}
			pkg.GoFiles = append(pkg.GoFiles, pkg.TestGoFiles...)
			pkg.PkgObj = "" // do not load from disk
			if err := api.BuildPackage(pkg); err != nil {
				return err
			}

			testPkg := &api.Package{Package: &build.Package{
				ImportPath: pkg.ImportPath + "_test",
				Dir:        pkg.Dir,
				GoFiles:    pkg.XTestGoFiles,
			}}
			if err := api.BuildPackage(testPkg); err != nil {
				return err
			}

			var names []string
			var tests []string
			imports := mainPkgTypes.Imports()
			collectTests := func(pkg *api.Package) {
				pkgTypes := api.TypesConfig.Packages[pkg.ImportPath]
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

		if err := api.WriteCommandPackage(mainPkg, tempfile.Name()); err != nil {
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
				if err := api.BuildFiles([]string{toolFlags.Arg(0)}, basename[:len(basename)-3]+".js"); err != nil {
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

func runNode(script string, args []string, dir string) error {
	node := exec.Command("node", append([]string{script}, args...)...)
	node.Dir = dir
	node.Stdin = os.Stdin
	node.Stdout = os.Stdout
	node.Stderr = os.Stderr
	return node.Run()
}
