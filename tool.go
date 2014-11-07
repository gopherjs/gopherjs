package main

import (
	"bytes"
	"fmt"
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

	"code.google.com/p/go.crypto/ssh/terminal"
	"code.google.com/p/go.tools/go/types"
	gbuild "github.com/gopherjs/gopherjs/build"
	"github.com/gopherjs/gopherjs/compiler"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
	options := &gbuild.Options{CreateMapFile: true}
	var pkgObj string

	pflag.BoolVarP(&options.Verbose, "verbose", "v", false, "print the names of packages as they are compiled")
	flagVerbose := pflag.Lookup("verbose")
	pflag.BoolVarP(&options.Watch, "watch", "w", false, "watch for changes to the source files")
	flagWatch := pflag.Lookup("watch")
	pflag.BoolVarP(&options.Minify, "minify", "m", false, "minify generated code")
	flagMinify := pflag.Lookup("minify")
	pflag.BoolVar(&options.Color, "color", terminal.IsTerminal(2) && os.Getenv("TERM") != "dumb", "colored output")
	flagColor := pflag.Lookup("color")

	cmdBuild := &cobra.Command{
		Use:   "build [packages]",
		Short: "compile packages and dependencies",
	}
	cmdBuild.Flags().StringVarP(&pkgObj, "output", "o", "", "output file")
	cmdBuild.Flags().AddFlag(flagVerbose)
	cmdBuild.Flags().AddFlag(flagWatch)
	cmdBuild.Flags().AddFlag(flagMinify)
	cmdBuild.Flags().AddFlag(flagColor)
	cmdBuild.Run = func(cmd *cobra.Command, args []string) {
		for {
			s := gbuild.NewSession(options)

			exitCode := handleError(func() error {
				if len(args) == 0 {
					return s.BuildDir(currentDirectory, currentDirectory, pkgObj)
				}

				if strings.HasSuffix(args[0], ".go") || strings.HasSuffix(args[0], ".inc.js") {
					for _, arg := range args {
						if !strings.HasSuffix(arg, ".go") && !strings.HasSuffix(arg, ".inc.js") {
							return fmt.Errorf("named files must be .go or .inc.js files")
						}
					}
					if pkgObj == "" {
						basename := filepath.Base(args[0])
						pkgObj = basename[:len(basename)-3] + ".js"
					}
					names := make([]string, len(args))
					for i, name := range args {
						name = filepath.ToSlash(name)
						names[i] = name
						if s.Watcher != nil {
							s.Watcher.Add(filepath.ToSlash(name))
						}
					}
					if err := s.BuildFiles(args, pkgObj, currentDirectory); err != nil {
						return err
					}
					return nil
				}

				for _, pkgPath := range args {
					pkgPath = filepath.ToSlash(pkgPath)
					if s.Watcher != nil {
						s.Watcher.Add(pkgPath)
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
						pkgObj = filepath.Base(args[0]) + ".js"
					}
					if err := s.WriteCommandPackage(pkg, pkgObj); err != nil {
						return err
					}
				}
				return nil
			}, options)

			if s.Watcher == nil {
				os.Exit(exitCode)
			}
			s.WaitForChange()
		}
	}

	cmdInstall := &cobra.Command{
		Use:   "install [packages]",
		Short: "compile and install packages and dependencies",
	}
	cmdInstall.Flags().AddFlag(flagVerbose)
	cmdInstall.Flags().AddFlag(flagWatch)
	cmdInstall.Flags().AddFlag(flagMinify)
	cmdInstall.Flags().AddFlag(flagColor)
	cmdInstall.Run = func(cmd *cobra.Command, args []string) {
		for {
			s := gbuild.NewSession(options)

			exitCode := handleError(func() error {
				pkgs := args
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
				if cmd.Name() == "get" {
					goGet := exec.Command("go", append([]string{"get", "-d"}, pkgs...)...)
					goGet.Stdout = os.Stdout
					goGet.Stderr = os.Stderr
					if err := goGet.Run(); err != nil {
						return err
					}
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
			}, options)

			if s.Watcher == nil {
				os.Exit(exitCode)
			}
			s.WaitForChange()
		}
	}

	cmdGet := &cobra.Command{
		Use:   "get [packages]",
		Short: "download and install packages and dependencies",
	}
	cmdGet.Flags().AddFlag(flagVerbose)
	cmdGet.Flags().AddFlag(flagWatch)
	cmdGet.Flags().AddFlag(flagMinify)
	cmdGet.Flags().AddFlag(flagColor)
	cmdGet.Run = cmdInstall.Run

	cmdRun := &cobra.Command{
		Use:   "run [gofiles...] [arguments...]",
		Short: "compile and run Go program",
	}
	cmdRun.Run = func(cmd *cobra.Command, args []string) {
		os.Exit(handleError(func() error {
			lastSourceArg := 0
			for {
				if lastSourceArg == len(args) || !(strings.HasSuffix(args[lastSourceArg], ".go") || strings.HasSuffix(args[lastSourceArg], ".inc.js")) {
					break
				}
				lastSourceArg++
			}
			if lastSourceArg == 0 {
				return fmt.Errorf("gopherjs run: no go files listed")
			}

			tempfile, err := ioutil.TempFile("", filepath.Base(args[0])+".")
			if err != nil {
				return err
			}
			defer func() {
				tempfile.Close()
				os.Remove(tempfile.Name())
			}()
			s := gbuild.NewSession(options)
			if err := s.BuildFiles(args[:lastSourceArg], tempfile.Name(), currentDirectory); err != nil {
				return err
			}
			if err := runNode(tempfile.Name(), args[lastSourceArg:], ""); err != nil {
				return err
			}
			return nil
		}, options))
	}

	cmdTest := &cobra.Command{
		Use:   "test [packages]",
		Short: "test packages",
	}
	verbose := cmdTest.Flags().BoolP("verbose", "v", false, "verbose")
	short := cmdTest.Flags().Bool("short", false, "short")
	cmdTest.Flags().AddFlag(flagMinify)
	cmdTest.Flags().AddFlag(flagColor)
	cmdTest.Run = func(cmd *cobra.Command, args []string) {
		os.Exit(handleError(func() error {
			pkgs := make([]*build.Package, len(args))
			for i, pkgPath := range args {
				pkgPath = filepath.ToSlash(pkgPath)
				var err error
				pkgs[i], err = gbuild.Import(pkgPath, 0, "js")
				if err != nil {
					return err
				}
			}
			if len(pkgs) == 0 {
				firstGopathWorkspace := filepath.SplitList(build.Default.GOPATH)[0]
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
		}, options))
	}

	cmdTool := &cobra.Command{
		Use:   "tool [command] [args...]",
		Short: "run specified go tool",
	}
	cmdTool.Flags().BoolP("e", "e", false, "")
	cmdTool.Flags().BoolP("l", "l", false, "")
	cmdTool.Flags().BoolP("m", "m", false, "")
	cmdTool.Flags().StringP("o", "o", "", "")
	cmdTool.Flags().StringP("D", "D", "", "")
	cmdTool.Flags().StringP("I", "I", "", "")
	cmdTool.Run = func(cmd *cobra.Command, args []string) {
		os.Exit(handleError(func() error {
			if len(args) == 2 {
				switch args[0][1] {
				case 'g':
					basename := filepath.Base(args[1])
					s := gbuild.NewSession(options)
					if err := s.BuildFiles([]string{args[1]}, basename[:len(basename)-3]+".js", currentDirectory); err != nil {
						return err
					}
					return nil
				}
			}
			cmdTool.Help()
			return nil
		}, options))
	}

	rootCmd := &cobra.Command{
		Use:  "gopherjs",
		Long: "GopherJS is a tool for compiling Go source code to JavaScript.",
	}
	rootCmd.AddCommand(cmdBuild, cmdGet, cmdInstall, cmdRun, cmdTest, cmdTool)
	rootCmd.Execute()
}

func handleError(f func() error, options *gbuild.Options) int {
	switch err := f().(type) {
	case nil:
		return 0
	case compiler.ErrorList:
		for _, entry := range err {
			printError(entry, options)
		}
		return 1
	case *exec.ExitError:
		return err.Sys().(syscall.WaitStatus).ExitStatus()
	default:
		printError(err, options)
		return 1
	}
}

func printError(err error, options *gbuild.Options) {
	makeRel := func(name string) string {
		if relname, err := filepath.Rel(currentDirectory, name); err == nil {
			return relname
		}
		return name
	}

	switch e := err.(type) {
	case *scanner.Error:
		options.PrintError("%s:%d:%d: %s\n", makeRel(e.Pos.Filename), e.Pos.Line, e.Pos.Column, e.Msg)
	case types.Error:
		pos := e.Fset.Position(e.Pos)
		options.PrintError("%s:%d:%d: %s\n", makeRel(pos.Filename), pos.Line, pos.Column, e.Msg)
	default:
		options.PrintError("%s\n", e)
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
