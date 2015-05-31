package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"
	"time"

	gbuild "github.com/gopherjs/gopherjs/build"
	"github.com/gopherjs/gopherjs/compiler"
	"github.com/neelance/sourcemap"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/tools/go/types"
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
	tags := pflag.String("tags", "", "a list of build tags to consider satisfied during the build")
	flagTags := pflag.Lookup("tags")

	cmdBuild := &cobra.Command{
		Use:   "build [packages]",
		Short: "compile packages and dependencies",
	}
	cmdBuild.Flags().StringVarP(&pkgObj, "output", "o", "", "output file")
	cmdBuild.Flags().AddFlag(flagVerbose)
	cmdBuild.Flags().AddFlag(flagWatch)
	cmdBuild.Flags().AddFlag(flagMinify)
	cmdBuild.Flags().AddFlag(flagColor)
	cmdBuild.Flags().AddFlag(flagTags)
	cmdBuild.Run = func(cmd *cobra.Command, args []string) {
		options.BuildTags = strings.Fields(*tags)
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
							s.Watcher.Add(name)
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
					buildPkg, err := gbuild.Import(pkgPath, 0, s.InstallSuffix(), options.BuildTags)
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
	cmdInstall.Flags().AddFlag(flagTags)
	cmdInstall.Run = func(cmd *cobra.Command, args []string) {
		options.BuildTags = strings.Fields(*tags)
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
	cmdGet.Flags().AddFlag(flagTags)
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
				os.Remove(tempfile.Name() + ".map")
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
	bench := cmdTest.Flags().String("bench", "", "Run benchmarks matching the regular expression. By default, no benchmarks run. To run all benchmarks, use '--bench=.'.")
	run := cmdTest.Flags().String("run", "", "Run only those tests and examples matching the regular expression.")
	short := cmdTest.Flags().Bool("short", false, "Tell long-running tests to shorten their run time.")
	verbose := cmdTest.Flags().BoolP("verbose", "v", false, "Log all tests as they are run. Also print all text from Log and Logf calls even if the test succeeds.")
	cmdTest.Flags().AddFlag(flagMinify)
	cmdTest.Flags().AddFlag(flagColor)
	cmdTest.Run = func(cmd *cobra.Command, args []string) {
		os.Exit(handleError(func() error {
			pkgs := make([]*build.Package, len(args))
			for i, pkgPath := range args {
				pkgPath = filepath.ToSlash(pkgPath)
				var err error
				pkgs[i], err = gbuild.Import(pkgPath, 0, "", nil)
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
					if pkg, err = gbuild.Import(pkgPath, 0, "", nil); err != nil {
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
			for _, pkg := range pkgs {
				if len(pkg.TestGoFiles) == 0 && len(pkg.XTestGoFiles) == 0 {
					fmt.Printf("?   \t%s\t[no test files]\n", pkg.ImportPath)
					continue
				}

				s := gbuild.NewSession(options)
				tests := &testFuncs{Package: pkg}
				collectTests := func(buildPkg *build.Package, testPkgName string, needVar *bool) error {
					testPkg := &gbuild.PackageData{Package: buildPkg}
					if err := s.BuildPackage(testPkg); err != nil {
						return err
					}

					for _, decl := range testPkg.Archive.Declarations {
						if strings.HasPrefix(decl.FullName, testPkg.ImportPath+".Test") {
							tests.Tests = append(tests.Tests, testFunc{Package: testPkgName, Name: decl.FullName[len(testPkg.ImportPath)+1:]})
							*needVar = true
						}
						if strings.HasPrefix(decl.FullName, testPkg.ImportPath+".Benchmark") {
							tests.Benchmarks = append(tests.Benchmarks, testFunc{Package: testPkgName, Name: decl.FullName[len(testPkg.ImportPath)+1:]})
							*needVar = true
						}
					}
					return nil
				}

				if err := collectTests(&build.Package{
					ImportPath: pkg.ImportPath,
					Dir:        pkg.Dir,
					GoFiles:    append(pkg.GoFiles, pkg.TestGoFiles...),
					Imports:    append(pkg.Imports, pkg.TestImports...),
				}, "_test", &tests.NeedTest); err != nil {
					return err
				}

				if err := collectTests(&build.Package{
					ImportPath: pkg.ImportPath + "_test",
					Dir:        pkg.Dir,
					GoFiles:    pkg.XTestGoFiles,
					Imports:    pkg.XTestImports,
				}, "_xtest", &tests.NeedXtest); err != nil {
					return err
				}

				buf := bytes.NewBuffer(nil)
				if err := testmainTmpl.Execute(buf, tests); err != nil {
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
					os.Remove(tempfile.Name() + ".map")
				}()

				if err := s.WriteCommandPackage(mainPkg, tempfile.Name()); err != nil {
					return err
				}

				var args []string
				if *bench != "" {
					args = append(args, "-test.bench", *bench)
				}
				if *run != "" {
					args = append(args, "-test.run", *run)
				}
				if *short {
					args = append(args, "-test.short")
				}
				if *verbose {
					args = append(args, "-test.v")
				}
				status := "ok  "
				start := time.Now()
				if err := runNode(tempfile.Name(), args, pkg.Dir); err != nil {
					if _, ok := err.(*exec.ExitError); !ok {
						return err
					}
					exitErr = err
					status = "FAIL"
				}
				fmt.Printf("%s\t%s\t%.3fs\n", status, pkg.ImportPath, time.Now().Sub(start).Seconds())
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

	cmdServe := &cobra.Command{
		Use:   "serve",
		Short: "compile on-the-fly and serve",
	}
	cmdServe.Flags().AddFlag(flagVerbose)
	cmdServe.Flags().AddFlag(flagMinify)
	cmdServe.Flags().AddFlag(flagColor)
	cmdServe.Flags().AddFlag(flagTags)
	var addr string
	cmdServe.Flags().StringVarP(&addr, "http", "", ":8080", "HTTP bind address to serve")
	cmdServe.Run = func(cmd *cobra.Command, args []string) {
		dirs := append(filepath.SplitList(build.Default.GOPATH), build.Default.GOROOT)
		sourceFiles := http.FileServer(serveCommandFileSystem{options: options, dirs: dirs, sourceMaps: make(map[string][]byte)})
		printServingAt(addr)
		fmt.Fprintln(os.Stderr, http.ListenAndServe(addr, sourceFiles))
	}

	rootCmd := &cobra.Command{
		Use:  "gopherjs",
		Long: "GopherJS is a tool for compiling Go source code to JavaScript.",
	}
	rootCmd.AddCommand(cmdBuild, cmdGet, cmdInstall, cmdRun, cmdTest, cmdTool, cmdServe)
	rootCmd.Execute()
}

type serveCommandFileSystem struct {
	options    *gbuild.Options
	dirs       []string
	sourceMaps map[string][]byte
}

func (fs serveCommandFileSystem) Open(name string) (http.File, error) {
	for _, d := range fs.dirs {
		file, err := http.Dir(filepath.Join(d, "src")).Open(name)
		if err == nil {
			return file, nil
		}
	}

	if strings.HasSuffix(name, "/main.js.map") {
		if content, ok := fs.sourceMaps[name]; ok {
			return newFakeFile("main.js.map", content), nil
		}
	}

	isIndex := strings.HasSuffix(name, "/index.html")
	isMain := strings.HasSuffix(name, "/main.js")
	if isIndex || isMain {
		s := gbuild.NewSession(fs.options)
		buildPkg, err := gbuild.Import(path.Dir(name[1:]), 0, s.InstallSuffix(), fs.options.BuildTags)
		if err != nil || buildPkg.Name != "main" {
			return nil, os.ErrNotExist
		}

		if isIndex {
			return newFakeFile("index.html", []byte(`<html><head><meta charset="utf-8"><script src="main.js"></script></head></html>`)), nil
		}

		if isMain {
			buf := bytes.NewBuffer(nil)
			handleError(func() error {
				pkg := &gbuild.PackageData{Package: buildPkg}
				if err := s.BuildPackage(pkg); err != nil {
					return err
				}

				sourceMapFilter := &compiler.SourceMapFilter{Writer: buf}
				m := &sourcemap.Map{File: "main.js"}
				sourceMapFilter.MappingCallback = gbuild.NewMappingCallback(m, fs.options.GOROOT, fs.options.GOPATH)

				deps, err := compiler.ImportDependencies(pkg.Archive, s.ImportContext.Import)
				if err != nil {
					return err
				}
				if err := compiler.WriteProgramCode(deps, sourceMapFilter); err != nil {
					return err
				}

				mapBuf := bytes.NewBuffer(nil)
				m.WriteTo(mapBuf)
				buf.WriteString("//# sourceMappingURL=main.js.map\n")
				fs.sourceMaps[name+".map"] = mapBuf.Bytes()

				return nil
			}, fs.options)
			return newFakeFile("main.js", buf.Bytes()), nil
		}
	}

	return nil, os.ErrNotExist
}

type fakeFile struct {
	name string
	size int
	io.ReadSeeker
}

func newFakeFile(name string, content []byte) *fakeFile {
	return &fakeFile{name: name, size: len(content), ReadSeeker: bytes.NewReader(content)}
}

func (f *fakeFile) Close() error {
	return nil
}

func (f *fakeFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, os.ErrInvalid
}

func (f *fakeFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *fakeFile) Name() string {
	return f.name
}

func (f *fakeFile) Size() int64 {
	return int64(f.size)
}

func (f *fakeFile) Mode() os.FileMode {
	return 0
}

func (f *fakeFile) ModTime() time.Time {
	return time.Time{}
}

func (f *fakeFile) IsDir() bool {
	return false
}

func (f *fakeFile) Sys() interface{} {
	return nil
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

// printServingAt prints "serving at:" message with all addresses where served content is available.
func printServingAt(addr string) {
	var hosts []string
	if len(addr) >= 1 && (addr)[0] == ':' { // ":port" form.
		ips, err := getAllIps()
		if err != nil {
			fmt.Fprintln(os.Stderr, "unable to get ips:", err)
			fmt.Printf("serving at %s\n", addr)
			return
		}
		for _, ip := range ips {
			if ip == "127.0.0.1" {
				ip = "localhost"
			}
			hosts = append(hosts, ip+addr)
		}
	} else { // "host" or "host:port" form.
		hosts = []string{addr}
	}
	fmt.Println("serving at:")
	for _, host := range hosts {
		fmt.Printf("http://%s\n", host)
	}
}

// getAllIps returns a string slice of all IPs.
func getAllIps() (ips []string, err error) {
	ifis, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, ifi := range ifis {
		addrs, err := ifi.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip4 := ipNet.IP.To4()
			if ip4 == nil {
				continue
			}
			ips = append(ips, ipNet.IP.String())
		}
	}
	return ips, nil
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
	"regexp"
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

var matchPat string
var matchRe *regexp.Regexp

func matchString(pat, str string) (result bool, err error) {
	if matchRe == nil || matchPat != pat {
		matchPat = pat
		matchRe, err = regexp.Compile(matchPat)
		if err != nil {
			return
		}
	}
	return matchRe.MatchString(str), nil
}

func main() {
	testing.Main(matchString, tests, benchmarks, examples)
}

`))
