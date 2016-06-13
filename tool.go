// GopherJS is a tool for compiling Go source code to JavaScript.
package main

import (
	"bytes"
	"fmt"
	"go/build"
	"go/scanner"
	"go/types"
	"io"
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
)

func main() {
	options := &gbuild.Options{CreateMapFile: true}
	var pkgObj string

	pflag.BoolVarP(&options.Verbose, "verbose", "v", false, "print the names of packages as they are compiled")
	flagVerbose := pflag.Lookup("verbose")
	pflag.BoolVarP(&options.Quiet, "quiet", "q", false, "suppress non-fatal warnings")
	flagQuiet := pflag.Lookup("quiet")
	pflag.BoolVarP(&options.Minify, "minify", "m", false, "minify generated code")
	flagMinify := pflag.Lookup("minify")
	pflag.BoolVar(&options.Color, "color", terminal.IsTerminal(int(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb", "colored output")
	flagColor := pflag.Lookup("color")
	tags := pflag.String("tags", "", "a list of build tags to consider satisfied during the build")
	flagTags := pflag.Lookup("tags")

	cmdBuild := &cobra.Command{
		Use:   "build [packages]",
		Short: "compile packages and dependencies",
	}
	cmdBuild.Flags().StringVarP(&pkgObj, "output", "o", "", "output file")
	cmdBuild.Flags().AddFlag(flagVerbose)
	cmdBuild.Flags().AddFlag(flagQuiet)
	cmdBuild.Flags().AddFlag(flagMinify)
	cmdBuild.Flags().AddFlag(flagColor)
	cmdBuild.Flags().AddFlag(flagTags)
	cmdBuild.Run = func(cmd *cobra.Command, args []string) {
		options.BuildTags = strings.Fields(*tags)

		build.Default.GOARCH = "js"
		cmdgo_buildContext = build.Default

		os.Args = []string{"gopherjs", "build", "-compiler=gopherjs"}
		if pkgObj != "" {
			os.Args = append(os.Args, "-o", pkgObj)
		}
		if options.Verbose {
			os.Args = append(os.Args, "-v")
		}
		if *tags != "" {
			os.Args = append(os.Args, "-tags", *tags)
		}
		os.Args = append(os.Args, args...)

		cmdgo_main()
	}

	cmdInstall := &cobra.Command{
		Use:   "install [packages]",
		Short: "compile and install packages and dependencies",
	}
	cmdInstall.Flags().AddFlag(flagVerbose)
	cmdInstall.Flags().AddFlag(flagQuiet)
	cmdInstall.Flags().AddFlag(flagMinify)
	cmdInstall.Flags().AddFlag(flagColor)
	cmdInstall.Flags().AddFlag(flagTags)
	cmdInstall.Run = func(cmd *cobra.Command, args []string) {
		options.BuildTags = strings.Fields(*tags)

		build.Default.GOARCH = "js"
		cmdgo_buildContext = build.Default

		os.Args = []string{"gopherjs", "install", "-compiler=gopherjs"}
		if options.Verbose {
			os.Args = append(os.Args, "-v")
		}
		if *tags != "" {
			os.Args = append(os.Args, "-tags", *tags)
		}
		os.Args = append(os.Args, args...)

		cmdgo_main()
	}

	cmdGet := &cobra.Command{
		Use:   "get [packages]",
		Short: "download and install packages and dependencies",
	}
	cmdGet.Flags().AddFlag(flagVerbose)
	cmdGet.Flags().AddFlag(flagQuiet)
	cmdGet.Flags().AddFlag(flagMinify)
	cmdGet.Flags().AddFlag(flagColor)
	cmdGet.Flags().AddFlag(flagTags)
	cmdGet.Run = func(_ *cobra.Command, args []string) {
		options.BuildTags = strings.Fields(*tags)

		build.Default.GOARCH = "js"
		cmdgo_buildContext = build.Default

		os.Args = []string{"gopherjs", "get", "-compiler=gopherjs"}
		if options.Verbose {
			os.Args = append(os.Args, "-v")
		}
		if *tags != "" {
			os.Args = append(os.Args, "-tags", *tags)
		}
		os.Args = append(os.Args, args...)

		cmdgo_main()
	}

	cmdRun := &cobra.Command{
		Use:   "run [gofiles...] [arguments...]",
		Short: "compile and run Go program",
	}
	cmdRun.Flags().AddFlag(flagVerbose)
	cmdRun.Run = func(cmd *cobra.Command, args []string) {
		options.BuildTags = strings.Fields(*tags)

		build.Default.GOARCH = "js"
		cmdgo_buildContext = build.Default

		os.Args = []string{"gopherjs", "run", "-compiler=gopherjs", "-exec=node"}
		if options.Verbose {
			os.Args = append(os.Args, "-v")
		}
		os.Args = append(os.Args, args...)

		cmdgo_main()
	}

	cmdTest := &cobra.Command{
		Use:   "test [packages]",
		Short: "test packages",
	}
	bench := cmdTest.Flags().String("bench", "", "Run benchmarks matching the regular expression. By default, no benchmarks run. To run all benchmarks, use '--bench=.'.")
	run := cmdTest.Flags().String("run", "", "Run only those tests and examples matching the regular expression.")
	short := cmdTest.Flags().Bool("short", false, "Tell long-running tests to shorten their run time.")
	verbose := cmdTest.Flags().BoolP("verbose", "v", false, "Log all tests as they are run. Also print all text from Log and Logf calls even if the test succeeds.")
	compileOnly := cmdTest.Flags().BoolP("compileonly", "c", false, "Compile the test binary to pkg.test.js but do not run it (where pkg is the last element of the package's import path). The file name can be changed with the -o flag.")
	outputFilename := cmdTest.Flags().StringP("output", "o", "", "Compile the test binary to the named file. The test still runs (unless -c is specified).")
	cmdTest.Flags().AddFlag(flagMinify)
	cmdTest.Flags().AddFlag(flagColor)
	cmdTest.Flags().AddFlag(flagTags)
	cmdTest.Run = func(_ *cobra.Command, args []string) {
		options.BuildTags = strings.Fields(*tags)

		build.Default.GOARCH = "js"
		cmdgo_buildContext = build.Default

		os.Args = []string{"gopherjs", "test", "-compiler=gopherjs", "-exec=node"}
		if *bench != "" {
			os.Args = append(os.Args, "-bench", *bench)
		}
		if *run != "" {
			os.Args = append(os.Args, "-run", *run)
		}
		if *short {
			os.Args = append(os.Args, "-short")
		}
		if *verbose {
			os.Args = append(os.Args, "-v")
		}
		if *compileOnly {
			os.Args = append(os.Args, "-c")
		}
		if *outputFilename != "" {
			os.Args = append(os.Args, "-o", *outputFilename)
		}
		if *tags != "" {
			os.Args = append(os.Args, "-tags", *tags)
		}
		os.Args = append(os.Args, args...)

		cmdgo_main()
	}

	cmdServe := &cobra.Command{
		Use:   "serve [root]",
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
		var root string

		if len(args) > 1 {
			cmdServe.Help()
			return
		}

		if len(args) == 1 {
			root = args[0]
		}

		sourceFiles := http.FileServer(serveCommandFileSystem{
			serveRoot:  root,
			options:    options,
			dirs:       dirs,
			sourceMaps: make(map[string][]byte),
		})

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
	rootCmd.AddCommand(cmdBuild, cmdGet, cmdInstall, cmdRun, cmdTest, cmdServe)
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
	serveRoot  string
	options    *gbuild.Options
	dirs       []string
	sourceMaps map[string][]byte
}

func (fs serveCommandFileSystem) Open(requestName string) (http.File, error) {
	name := path.Join(fs.serveRoot, requestName[1:]) // requestName[0] == '/'

	dir, file := path.Split(name)
	base := path.Base(dir) // base is parent folder name, which becomes the output file name.

	isPkg := file == base+".js"
	isMap := file == base+".js.map"
	isIndex := file == "index.html"

	if isPkg || isMap || isIndex {
		// If we're going to be serving our special files, make sure there's a Go command in this folder.
		s := gbuild.NewSession(fs.options)
		pkg, err := gbuild.Import(path.Dir(name), 0, s.InstallSuffix(), fs.options.BuildTags)
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
				archive, err := s.BuildPackage(pkg)
				if err != nil {
					return err
				}

				sourceMapFilter := &compiler.SourceMapFilter{Writer: buf}
				m := &sourcemap.Map{File: base + ".js"}
				sourceMapFilter.MappingCallback = gbuild.NewMappingCallback(m, fs.options.GOROOT, fs.options.GOPATH)

				deps, err := compiler.ImportDependencies(archive, s.BuildImportPath)
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
		dir := http.Dir(filepath.Join(d, "src"))

		f, err := dir.Open(name)
		if err == nil {
			return f, nil
		}

		// source maps are served outside of serveRoot
		f, err = dir.Open(requestName)
		if err == nil {
			return f, nil
		}
	}

	if isIndex {
		// If there was no index.html file in any dirs, supply our own.
		return newFakeFile("index.html", []byte(`<html><head><meta charset="utf-8"><script src="`+base+`.js"></script></head><body></body></html>`)), nil
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

// printError prints err to Stderr with options. If browserErrors is non-nil, errors are also written for presentation in browser.
func printError(err error, options *gbuild.Options, browserErrors *bytes.Buffer) {
	e := sprintError(err)
	options.PrintError("%s\n", e)
	if browserErrors != nil {
		fmt.Fprintln(browserErrors, `console.error("`+template.JSEscapeString(e)+`");`)
	}
}

// sprintError returns an annotated error string without trailing newline.
func sprintError(err error) string {
	makeRel := func(name string) string {
		if relname, err := filepath.Rel(cmdgo_cwd, name); err == nil {
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
