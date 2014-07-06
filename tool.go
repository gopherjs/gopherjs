package main

import (
	"bytes"
	"code.google.com/p/go.tools/go/types"
	"flag"
	"fmt"
	gbuild "github.com/gopherjs/gopherjs/build"
	"github.com/gopherjs/gopherjs/compiler"
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
	"text/template"
	"time"
)

var currentDirectory string

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

func main() {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	cmd := "help"
	var cmdArgs []string
	if err := flags.Parse(os.Args[1:]); err == nil && flags.NArg() != 0 {
		cmd = flags.Arg(0)
		cmdArgs = flags.Args()[1:]
		if cmd == "help" && flags.NArg() == 2 {
			cmd = flags.Arg(1)
			cmdArgs = []string{"--help"}
		}
	}

	options := &gbuild.Options{CreateMapFile: true}
	switch cmd {
	case "build":
		buildFlags := flag.NewFlagSet("build command", flag.ExitOnError)
		var pkgObj string
		buildFlags.StringVar(&pkgObj, "o", "", "output file")
		buildFlags.BoolVar(&options.Verbose, "v", false, "print the names of packages as they are compiled")
		buildFlags.BoolVar(&options.Watch, "w", false, "watch for changes to the source files")
		buildFlags.BoolVar(&options.Minify, "m", false, "minify generated code")
		buildFlags.Parse(cmdArgs)

		for {
			s := gbuild.NewSession(options)

			exitCode := handleError(func() error {
				if buildFlags.NArg() == 0 {
					return s.BuildDir(currentDirectory, currentDirectory, pkgObj)
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
					names := make([]string, buildFlags.NArg())
					for i, name := range buildFlags.Args() {
						name = filepath.ToSlash(name)
						names[i] = name
						if s.Watcher != nil {
							s.Watcher.Watch(filepath.ToSlash(name))
						}
					}
					if err := s.BuildFiles(buildFlags.Args(), pkgObj, currentDirectory); err != nil {
						return err
					}
					return nil
				}

				for _, pkgPath := range buildFlags.Args() {
					pkgPath = filepath.ToSlash(pkgPath)
					if s.Watcher != nil {
						s.Watcher.Watch(pkgPath)
					}
					buildPkg, err := gbuild.Import(pkgPath, 0, s.ArchSuffix())
					if err != nil {
						return err
					}
					pkg := &gbuild.PackageData{Package: buildPkg}
					if err := s.BuildPackage(pkg); err != nil {
						return err
					}
					if pkgObj == "" {
						pkgObj = filepath.Base(buildFlags.Arg(0)) + ".js"
					}
					if err := s.WriteCommandPackage(pkg, pkgObj); err != nil {
						return err
					}
				}
				return nil
			})

			if s.Watcher == nil {
				os.Exit(exitCode)
			}
			s.WaitForChange()
		}

	case "install":
		installFlags := flag.NewFlagSet("install command", flag.ExitOnError)
		installFlags.BoolVar(&options.Verbose, "v", false, "print the names of packages as they are compiled")
		installFlags.BoolVar(&options.Watch, "w", false, "watch for changes to the source files")
		installFlags.BoolVar(&options.Minify, "m", false, "minify generated code")
		installFlags.Parse(cmdArgs)

		for {
			s := gbuild.NewSession(options)

			exitCode := handleError(func() error {
				pkgs := installFlags.Args()
				if len(pkgs) == 0 {
					firstGopathWorkspace := filepath.SplitList(build.Default.GOPATH)[0] // TODO: The GOPATH workspace that contains the package source should be chosen.
					srcDir, err := filepath.EvalSymlinks(filepath.Join(firstGopathWorkspace, "src"))
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
					pkgPath = filepath.ToSlash(pkgPath)
					if _, err := s.ImportPackage(pkgPath); err != nil {
						return err
					}
					pkg := s.Packages[pkgPath]
					if err := s.WriteCommandPackage(pkg, pkg.PkgObj); err != nil {
						return err
					}
				}
				return nil
			})

			if s.Watcher == nil {
				os.Exit(exitCode)
			}
			s.WaitForChange()
		}

	case "run":
		runFlags := flag.NewFlagSet("run command", flag.ExitOnError)
		runFlags.Parse(cmdArgs)

		os.Exit(handleError(func() error {
			lastSourceArg := 0
			for {
				if !strings.HasSuffix(runFlags.Arg(lastSourceArg), ".go") {
					break
				}
				lastSourceArg++
			}
			if lastSourceArg == 0 {
				return fmt.Errorf("gopherjs run: no go files listed")
			}

			tempfile, err := ioutil.TempFile("", filepath.Base(runFlags.Arg(0))+".")
			if err != nil {
				return err
			}
			defer func() {
				tempfile.Close()
				os.Remove(tempfile.Name())
			}()
			s := gbuild.NewSession(options)
			if err := s.BuildFiles(runFlags.Args()[:lastSourceArg], tempfile.Name(), currentDirectory); err != nil {
				return err
			}
			if err := runNode(tempfile.Name(), runFlags.Args()[lastSourceArg:], ""); err != nil {
				return err
			}
			return nil
		}))

	case "test":
		testFlags := flag.NewFlagSet("test command", flag.ExitOnError)
		verbose := testFlags.Bool("v", false, "verbose")
		short := testFlags.Bool("short", false, "short")
		testFlags.BoolVar(&options.Minify, "m", false, "minify generated code")
		testFlags.Parse(cmdArgs)

		os.Exit(handleError(func() error {
			pkgs := make([]*build.Package, testFlags.NArg())
			for i, pkgPath := range testFlags.Args() {
				pkgPath = filepath.ToSlash(pkgPath)
				var err error
				pkgs[i], err = gbuild.Import(pkgPath, 0, "js")
				if err != nil {
					return err
				}
			}
			if len(pkgs) == 0 {
				firstGopathWorkspace := filepath.SplitList(build.Default.GOPATH)[0] // TODO: Not sure if always picking first GOPATH workspace here is the right thing.
				srcDir, err := filepath.EvalSymlinks(filepath.Join(firstGopathWorkspace, "src"))
				if err != nil {
					return err
				}
				var pkg *build.Package
				if strings.HasPrefix(currentDirectory, srcDir) {
					pkgPath, err := filepath.Rel(srcDir, currentDirectory)
					if err != nil {
						return err
					}
					if pkg, err = gbuild.Import(pkgPath, 0, "js"); err != nil {
						return err
					}
				}
				if pkg == nil {
					if pkg, err = build.ImportDir(currentDirectory, 0); err != nil {
						return err
					}
					pkg.ImportPath = "_" + currentDirectory
				}
				pkgs = []*build.Package{pkg}
			}

			var exitErr error
			for _, buildPkg := range pkgs {
				if len(buildPkg.TestGoFiles) == 0 && len(buildPkg.XTestGoFiles) == 0 {
					fmt.Printf("?   \t%s\t[no test files]\n", buildPkg.ImportPath)
					continue
				}

				buildPkg.PkgObj = ""
				buildPkg.GoFiles = append(buildPkg.GoFiles, buildPkg.TestGoFiles...)
				pkg := &gbuild.PackageData{Package: buildPkg}

				s := gbuild.NewSession(options)
				if err := s.BuildPackage(pkg); err != nil {
					return err
				}

				tests := &testFuncs{Package: buildPkg}
				collectTests := func(pkg *gbuild.PackageData, testPkg string) bool {
					for _, name := range pkg.Archive.Tests {
						tests.Tests = append(tests.Tests, testFunc{Package: testPkg, Name: name})
					}
					return len(pkg.Archive.Tests) != 0
				}

				tests.NeedTest = collectTests(pkg, "_test")
				if len(pkg.XTestGoFiles) != 0 {
					testPkg := &gbuild.PackageData{Package: &build.Package{
						ImportPath: pkg.ImportPath + "_test",
						Dir:        pkg.Dir,
						GoFiles:    pkg.XTestGoFiles,
					}}
					if err := s.BuildPackage(testPkg); err != nil {
						return err
					}
					tests.NeedXtest = collectTests(testPkg, "_xtest")
				}

				buf := bytes.NewBuffer(nil)
				err := testmainTmpl.Execute(buf, tests)
				if err != nil {
					return err
				}

				fset := token.NewFileSet()
				mainFile, err := parser.ParseFile(fset, "_testmain.go", buf, 0)
				if err != nil {
					return err
				}

				mainPkg := &gbuild.PackageData{
					Package: &build.Package{
						Name:       "main",
						ImportPath: "main",
					},
				}
				mainPkg.Archive, err = compiler.Compile("main", []*ast.File{mainFile}, fset, s.ImportContext, options.Minify)
				if err != nil {
					return err
				}

				tempfile, err := ioutil.TempFile("", "test.")
				if err != nil {
					return err
				}
				defer func() {
					tempfile.Close()
					os.Remove(tempfile.Name())
				}()

				if err := s.WriteCommandPackage(mainPkg, tempfile.Name()); err != nil {
					return err
				}

				var args []string
				if *verbose {
					args = append(args, "-test.v")
				}
				if *short {
					args = append(args, "-test.short")
				}
				status := "ok  "
				start := time.Now()
				if err := runNode(tempfile.Name(), args, buildPkg.Dir); err != nil {
					if _, ok := err.(*exec.ExitError); !ok {
						return err
					}
					exitErr = err
					status = "FAIL"
				}
				fmt.Printf("%s\t%s\t%.3fs\n", status, buildPkg.ImportPath, time.Now().Sub(start).Seconds())
			}
			return exitErr
		}))

	case "tool":
		tool := cmdArgs[0]
		toolFlags := flag.NewFlagSet("tool command", flag.ExitOnError)
		toolFlags.Bool("e", false, "")
		toolFlags.Bool("l", false, "")
		toolFlags.Bool("m", false, "")
		toolFlags.String("o", "", "")
		toolFlags.String("D", "", "")
		toolFlags.String("I", "", "")
		toolFlags.Parse(flags.Args()[2:])

		os.Exit(handleError(func() error {
			if len(tool) == 2 {
				switch tool[1] {
				case 'g':
					basename := filepath.Base(toolFlags.Arg(0))
					s := gbuild.NewSession(options)
					if err := s.BuildFiles([]string{toolFlags.Arg(0)}, basename[:len(basename)-3]+".js", currentDirectory); err != nil {
						return err
					}
					return nil
				}
			}
			return fmt.Errorf("Tool not supported: " + tool)
		}))

	case "help", "":
		os.Stderr.WriteString(`GopherJS is a tool for compiling Go source code to JavaScript.

Usage:

    gopherjs command [arguments]

The commands are:

    build       compile packages and dependencies
    install     compile and install packages and dependencies
    run         compile and run Go program (requires Node.js)
    test        test packages (requires Node.js)

Use "go help [command]" for more information about a command.

`)

	default:
		fmt.Fprintf(os.Stderr, "gopherjs: unknown subcommand \"%s\"\nRun 'gopherjs help' for usage.\n", cmd)

	}
}

func handleError(f func() error) int {
	switch err := f().(type) {
	case nil:
		return 0
	case compiler.ErrorList:
		makeRel := func(name string) string {
			if relname, err := filepath.Rel(currentDirectory, name); err == nil {
				if relname[0] != '.' {
					return "." + string(filepath.Separator) + relname
				}
				return relname
			}
			return name
		}
		for _, entry := range err {
			switch e := entry.(type) {
			case *scanner.Error:
				fmt.Fprintf(os.Stderr, "\x1B[31m%s:%d:%d: %s\x1B[39m\n", makeRel(e.Pos.Filename), e.Pos.Line, e.Pos.Column, e.Msg)
			case types.Error:
				pos := e.Fset.Position(e.Pos)
				fmt.Fprintf(os.Stderr, "\x1B[31m%s:%d:%d: %s\x1B[39m\n", makeRel(pos.Filename), pos.Line, pos.Column, e.Msg)
			default:
				fmt.Fprintf(os.Stderr, "\x1B[31m%s\x1B[39m\n", entry)
			}
		}
		return 1
	case *exec.ExitError:
		return err.Sys().(syscall.WaitStatus).ExitStatus()
	default:
		fmt.Fprintf(os.Stderr, "\x1B[31m%s\x1B[39m\n", err)
		return 1
	}
}

func runNode(script string, args []string, dir string) error {
	node := exec.Command("node", append([]string{script}, args...)...)
	node.Dir = dir
	node.Stdin = os.Stdin
	node.Stdout = os.Stdout
	node.Stderr = os.Stderr
	err := node.Run()
	if _, ok := err.(*exec.ExitError); err != nil && !ok {
		err = fmt.Errorf("could not run Node.js: %s", err.Error())
	}
	return err
}

type testFuncs struct {
	Tests      []testFunc
	Benchmarks []testFunc
	Examples   []testFunc
	Package    *build.Package
	NeedTest   bool
	NeedXtest  bool
}

type testFunc struct {
	Package string // imported package name (_test or _xtest)
	Name    string // function name
	Output  string // output, for examples
}

var testmainTmpl = template.Must(template.New("main").Parse(`
package main

import (
	"testing"

{{if .NeedTest}}
	_test {{.Package.ImportPath | printf "%q"}}
{{end}}
{{if .NeedXtest}}
	_xtest {{.Package.ImportPath | printf "%s_test" | printf "%q"}}
{{end}}
)

var tests = []testing.InternalTest{
{{range .Tests}}
	{"{{.Name}}", {{.Package}}.{{.Name}}},
{{end}}
}

var benchmarks = []testing.InternalBenchmark{
{{range .Benchmarks}}
	{"{{.Name}}", {{.Package}}.{{.Name}}},
{{end}}
}

var examples = []testing.InternalExample{
{{range .Examples}}
	{"{{.Name}}", {{.Package}}.{{.Name}}, {{.Output | printf "%q"}}},
{{end}}
}

func main() {
	testing.Main(nil, tests, benchmarks, examples)
}

`))
