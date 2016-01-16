package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
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
)

var currentDirectory string

func init() {
	var err error
	currentDirectory, err = os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	currentDirectory, err = filepath.EvalSymlinks(currentDirectory)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	gopaths := filepath.SplitList(build.Default.GOPATH)
	if len(gopaths) == 0 {
		fmt.Fprintf(os.Stderr, "$GOPATH not set. For more details see: go help gopath\n")
		os.Exit(1)
	}
}

func main() {
	options := &gbuild.Options{CreateMapFile: true}
	var pkgObj string

	pflag.BoolVarP(&options.Verbose, "verbose", "v", false, "print the names of packages as they are compiled")
	flagVerbose := pflag.Lookup("verbose")
	pflag.BoolVarP(&options.Quiet, "quiet", "q", false, "suppress non-fatal warnings")
	flagQuiet := pflag.Lookup("quiet")
	pflag.BoolVarP(&options.Watch, "watch", "w", false, "watch for changes to the source files")
	flagWatch := pflag.Lookup("watch")
	pflag.BoolVarP(&options.Minify, "minify", "m", false, "minify generated code")
	flagMinify := pflag.Lookup("minify")
	pflag.BoolVar(&options.Color, "color", terminal.IsTerminal(int(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb", "colored output")
	flagColor := pflag.Lookup("color")
	tags := pflag.String("tags", "", "a list of build tags to consider satisfied during the build")
	flagTags := pflag.Lookup("tags")
	jsCmd := pflag.String("exec", "", "specify alternate js interpreter (default: node)")

	cmdBuild := &cobra.Command{
		Use:   "build [packages]",
		Short: "compile packages and dependencies",
	}
	cmdBuild.Flags().StringVarP(&pkgObj, "output", "o", "", "output file")
	cmdBuild.Flags().AddFlag(flagVerbose)
	cmdBuild.Flags().AddFlag(flagQuiet)
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
					pkg, err := gbuild.Import(pkgPath, 0, s.InstallSuffix(), options.BuildTags)
					if err != nil {
						return err
					}
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
			}, options, nil)

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
	cmdInstall.Flags().AddFlag(flagQuiet)
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
					goGet := exec.Command("go", append([]string{"get", "-d", "-tags=js"}, pkgs...)...)
					goGet.Stdout = os.Stdout
					goGet.Stderr = os.Stderr
					if err := goGet.Run(); err != nil {
						return err
					}
				}
				for _, pkgPath := range pkgs {
					pkgPath = filepath.ToSlash(pkgPath)
					if _, err := s.BuildImportPath(pkgPath); err != nil {
						return err
					}
					pkg := s.Packages[pkgPath]
					if err := s.WriteCommandPackage(pkg, pkg.PkgObj); err != nil {
						return err
					}
				}
				return nil
			}, options, nil)

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
	cmdGet.Flags().AddFlag(flagQuiet)
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

			tempfile, err := ioutil.TempFile(currentDirectory, filepath.Base(args[0])+".")
			if err != nil && strings.HasPrefix(currentDirectory, runtime.GOROOT()) {
				tempfile, err = ioutil.TempFile("", filepath.Base(args[0])+".")
			}
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
			if err := runJS(tempfile.Name(), *jsCmd, args[lastSourceArg:], "", options.Quiet); err != nil {
				return err
			}
			return nil
		}, options, nil))
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
			pkgs := make([]*gbuild.PackageData, len(args))
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
				var pkg *gbuild.PackageData
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
					if pkg, err = gbuild.ImportDir(currentDirectory, 0); err != nil {
						return err
					}
					pkg.ImportPath = "_" + currentDirectory
				}
				pkgs = []*gbuild.PackageData{pkg}
			}

			var exitErr error
			for _, pkg := range pkgs {
				if len(pkg.TestGoFiles) == 0 && len(pkg.XTestGoFiles) == 0 {
					fmt.Printf("?   \t%s\t[no test files]\n", pkg.ImportPath)
					continue
				}

				s := gbuild.NewSession(options)
				tests := &testFuncs{Package: pkg.Package}
				collectTests := func(testPkg *gbuild.PackageData, testPkgName string, needVar *bool) error {
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

				if err := collectTests(&gbuild.PackageData{
					Package: &build.Package{
						ImportPath: pkg.ImportPath,
						Dir:        pkg.Dir,
						GoFiles:    append(pkg.GoFiles, pkg.TestGoFiles...),
						Imports:    append(pkg.Imports, pkg.TestImports...),
					},
					IsTest:  true,
					JSFiles: pkg.JSFiles,
				}, "_test", &tests.NeedTest); err != nil {
					return err
				}

				if err := collectTests(&gbuild.PackageData{
					Package: &build.Package{
						ImportPath: pkg.ImportPath + "_test",
						Dir:        pkg.Dir,
						GoFiles:    pkg.XTestGoFiles,
						Imports:    pkg.XTestImports,
					},
					IsTest: true,
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

				tempfile, err := ioutil.TempFile(currentDirectory, "test.")
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
				if err := runJS(tempfile.Name(), *jsCmd, args, pkg.Dir, options.Quiet); err != nil {
					if _, ok := err.(*exec.ExitError); !ok {
						return err
					}
					exitErr = err
					status = "FAIL"
				}
				fmt.Printf("%s\t%s\t%.3fs\n", status, pkg.ImportPath, time.Now().Sub(start).Seconds())
			}
			return exitErr
		}, options, nil))
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
		}, options, nil))
	}

	cmdServe := &cobra.Command{
		Use:   "serve",
		Short: "compile on-the-fly and serve",
	}
	cmdServe.Flags().AddFlag(flagVerbose)
	cmdServe.Flags().AddFlag(flagQuiet)
	cmdServe.Flags().AddFlag(flagMinify)
	cmdServe.Flags().AddFlag(flagColor)
	cmdServe.Flags().AddFlag(flagTags)
	var addr string
	cmdServe.Flags().StringVarP(&addr, "http", "", ":8080", "HTTP bind address to serve")
	cmdServe.Run = func(cmd *cobra.Command, args []string) {
		options.BuildTags = strings.Fields(*tags)
		dirs := append(filepath.SplitList(build.Default.GOPATH), build.Default.GOROOT)
		sourceFiles := http.FileServer(serveCommandFileSystem{options: options, dirs: dirs, sourceMaps: make(map[string][]byte)})
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if tcpAddr := ln.Addr().(*net.TCPAddr); tcpAddr.IP.Equal(net.IPv4zero) || tcpAddr.IP.Equal(net.IPv6zero) { // Any available addresses.
			fmt.Printf("serving at http://localhost:%d and on port %d of any available addresses\n", tcpAddr.Port, tcpAddr.Port)
		} else { // Specific address.
			fmt.Printf("serving at http://%s\n", tcpAddr)
		}
		fmt.Fprintln(os.Stderr, http.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)}, sourceFiles))
	}

	rootCmd := &cobra.Command{
		Use:  "gopherjs",
		Long: "GopherJS is a tool for compiling Go source code to JavaScript.",
	}
	rootCmd.AddCommand(cmdBuild, cmdGet, cmdInstall, cmdRun, cmdTest, cmdTool, cmdServe)
	rootCmd.Execute()
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

type serveCommandFileSystem struct {
	options    *gbuild.Options
	dirs       []string
	sourceMaps map[string][]byte
}

func (fs serveCommandFileSystem) Open(name string) (http.File, error) {
	dir, file := path.Split(name)
	base := path.Base(dir) // base is parent folder name, which becomes the output file name.

	isPkg := file == base+".js"
	isMap := file == base+".js.map"
	isIndex := file == "index.html"

	if isPkg || isMap || isIndex {
		// If we're going to be serving our special files, make sure there's a Go command in this folder.
		s := gbuild.NewSession(fs.options)
		pkg, err := gbuild.Import(path.Dir(name[1:]), 0, s.InstallSuffix(), fs.options.BuildTags)
		if err != nil || pkg.Name != "main" {
			isPkg = false
			isMap = false
			isIndex = false
		}

		switch {
		case isPkg:
			buf := bytes.NewBuffer(nil)
			browserErrors := bytes.NewBuffer(nil)
			exitCode := handleError(func() error {
				if err := s.BuildPackage(pkg); err != nil {
					return err
				}

				sourceMapFilter := &compiler.SourceMapFilter{Writer: buf}
				m := &sourcemap.Map{File: base + ".js"}
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
				buf.WriteString("//# sourceMappingURL=" + base + ".js.map\n")
				fs.sourceMaps[name+".map"] = mapBuf.Bytes()

				return nil
			}, fs.options, browserErrors)
			if exitCode != 0 {
				buf = browserErrors
			}
			return newFakeFile(base+".js", buf.Bytes()), nil

		case isMap:
			if content, ok := fs.sourceMaps[name]; ok {
				return newFakeFile(base+".js.map", content), nil
			}
		}
	}

	for _, d := range fs.dirs {
		f, err := http.Dir(filepath.Join(d, "src")).Open(name)
		if err == nil {
			return f, nil
		}
	}

	if isIndex {
		// If there was no index.html file in any dirs, supply our own.
		return newFakeFile("index.html", []byte(`<html><head><meta charset="utf-8"><script src="`+base+`.js"></script></head></html>`)), nil
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

// If browserErrors is non-nil, errors are written for presentation in browser.
func handleError(f func() error, options *gbuild.Options, browserErrors *bytes.Buffer) int {
	switch err := f().(type) {
	case nil:
		return 0
	case compiler.ErrorList:
		for _, entry := range err {
			printError(entry, options, browserErrors)
		}
		return 1
	case *exec.ExitError:
		return err.Sys().(syscall.WaitStatus).ExitStatus()
	default:
		printError(err, options, browserErrors)
		return 1
	}
}

// sprintError returns an annotated error string without trailing newline.
func sprintError(err error) string {
	makeRel := func(name string) string {
		if relname, err := filepath.Rel(currentDirectory, name); err == nil {
			return relname
		}
		return name
	}

	switch e := err.(type) {
	case *scanner.Error:
		return fmt.Sprintf("%s:%d:%d: %s", makeRel(e.Pos.Filename), e.Pos.Line, e.Pos.Column, e.Msg)
	case types.Error:
		pos := e.Fset.Position(e.Pos)
		return fmt.Sprintf("%s:%d:%d: %s", makeRel(pos.Filename), pos.Line, pos.Column, e.Msg)
	default:
		return fmt.Sprintf("%s", e)
	}
}

// printError prints err to Stderr with options. If browserErrors is non-nil, errors are also written for presentation in browser.
func printError(err error, options *gbuild.Options, browserErrors *bytes.Buffer) {
	e := sprintError(err)
	options.PrintError("%s\n", e)
	if browserErrors != nil {
		fmt.Fprintln(browserErrors, `console.error("`+template.JSEscapeString(e)+`");`)
	}
}

func runJS(script string, cmd string, args []string, dir string, quiet bool) error {
	var allArgs []string
	if len(cmd) == 0 {
		cmd = "node"
		// Use the default nodejs interpretor
		if b, _ := strconv.ParseBool(os.Getenv("SOURCE_MAP_SUPPORT")); os.Getenv("SOURCE_MAP_SUPPORT") == "" || b {
			allArgs = []string{"--require", "source-map-support/register"}
			if err := exec.Command(cmd, "--require", "source-map-support/register", "--eval", "").Run(); err != nil {
				if !quiet {
					fmt.Fprintln(os.Stderr, "gopherjs: Source maps disabled. Use Node.js 4.x with source-map-support module for nice stack traces.")
				}
				allArgs = []string{}
			}
		}

		if runtime.GOOS != "windows" {
			allArgs = append(allArgs, "--stack_size=10000", script)
		}
	} else {
		allArgs = []string{script}
	}

	allArgs = append(allArgs, args...)

	node := exec.Command(cmd, allArgs...)
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
