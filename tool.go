package main

import (
	"bytes"
	"errors"
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
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	gbuild "github.com/gopherjs/gopherjs/build"
	"github.com/gopherjs/gopherjs/build/cache"
	"github.com/gopherjs/gopherjs/compiler"
	"github.com/gopherjs/gopherjs/internal/sysutil"
	"github.com/neelance/sourcemap"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
	"golang.org/x/term"
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

	e := gbuild.DefaultEnv()
	if e.GOOS != "js" || e.GOARCH != "ecmascript" {
		fmt.Fprintf(os.Stderr, "Using GOOS=%s and GOARCH=%s in GopherJS is deprecated and will be removed in future. Use GOOS=js GOARCH=ecmascript instead.\n", e.GOOS, e.GOARCH)
	}
}

func main() {
	var (
		options = &gbuild.Options{}
		pkgObj  string
		tags    string
	)

	flagVerbose := pflag.NewFlagSet("", 0)
	flagVerbose.BoolVarP(&options.Verbose, "verbose", "v", false, "print the names of packages as they are compiled")
	flagQuiet := pflag.NewFlagSet("", 0)
	flagQuiet.BoolVarP(&options.Quiet, "quiet", "q", false, "suppress non-fatal warnings")

	compilerFlags := pflag.NewFlagSet("", 0)
	compilerFlags.BoolVarP(&options.Minify, "minify", "m", false, "minify generated code")
	compilerFlags.BoolVar(&options.Color, "color", term.IsTerminal(int(os.Stderr.Fd())) && os.Getenv("TERM") != "dumb", "colored output")
	compilerFlags.StringVar(&tags, "tags", "", "a list of build tags to consider satisfied during the build")
	compilerFlags.BoolVar(&options.MapToLocalDisk, "localmap", false, "use local paths for sourcemap")
	compilerFlags.BoolVarP(&options.NoCache, "no_cache", "a", false, "rebuild all packages from scratch")
	compilerFlags.BoolVarP(&options.CreateMapFile, "source_map", "s", true, "enable generation of source maps")

	flagWatch := pflag.NewFlagSet("", 0)
	flagWatch.BoolVarP(&options.Watch, "watch", "w", false, "watch for changes to the source files")

	cmdBuild := &cobra.Command{
		Use:   "build [packages]",
		Short: "compile packages and dependencies",
	}
	cmdBuild.Flags().StringVarP(&pkgObj, "output", "o", "", "output file")
	cmdBuild.Flags().AddFlagSet(flagVerbose)
	cmdBuild.Flags().AddFlagSet(flagQuiet)
	cmdBuild.Flags().AddFlagSet(compilerFlags)
	cmdBuild.Flags().AddFlagSet(flagWatch)
	cmdBuild.RunE = func(cmd *cobra.Command, args []string) error {
		options.BuildTags = strings.Fields(tags)
		for {
			s, err := gbuild.NewSession(options)
			if err != nil {
				options.PrintError("%s\n", err)
				return err
			}

			err = func() error {
				// Handle "gopherjs build [files]" ad-hoc package mode.
				if len(args) > 0 && (strings.HasSuffix(args[0], ".go") || strings.HasSuffix(args[0], ".inc.js")) {
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
					err := s.BuildFiles(args, pkgObj, currentDirectory)
					return err
				}

				xctx := gbuild.NewBuildContext(s.InstallSuffix(), options.BuildTags)
				// Expand import path patterns.
				pkgs, err := xctx.Match(args)
				if err != nil {
					return fmt.Errorf("failed to expand patterns %v: %w", args, err)
				}
				for _, pkgPath := range pkgs {
					if s.Watcher != nil {
						pkg, err := xctx.Import(pkgPath, currentDirectory, build.FindOnly)
						if err != nil {
							return err
						}
						s.Watcher.Add(pkg.Dir)
					}
					pkg, err := xctx.Import(pkgPath, currentDirectory, 0)
					if err != nil {
						return err
					}
					archive, err := s.BuildProject(pkg)
					if err != nil {
						return err
					}
					if len(pkgs) == 1 { // Only consider writing output if single package specified.
						if pkgObj == "" {
							pkgObj = filepath.Base(pkg.Dir) + ".js"
						}
						if pkg.IsCommand() && !pkg.UpToDate {
							if err := s.WriteCommandPackage(archive, pkgObj); err != nil {
								return err
							}
						}
					}
				}
				return nil
			}()

			if s.Watcher == nil {
				return err
			} else if err != nil {
				handleError(err, options, nil)
			}
			s.WaitForChange()
		}
	}

	cmdInstall := &cobra.Command{
		Use:   "install [packages]",
		Short: "compile and install packages and dependencies",
	}
	cmdInstall.Flags().AddFlagSet(flagVerbose)
	cmdInstall.Flags().AddFlagSet(flagQuiet)
	cmdInstall.Flags().AddFlagSet(compilerFlags)
	cmdInstall.Flags().AddFlagSet(flagWatch)
	cmdInstall.RunE = func(cmd *cobra.Command, args []string) error {
		options.BuildTags = strings.Fields(tags)
		for {
			s, err := gbuild.NewSession(options)
			if err != nil {
				return err
			}

			err = func() error {
				// Expand import path patterns.
				xctx := gbuild.NewBuildContext(s.InstallSuffix(), options.BuildTags)
				pkgs, err := xctx.Match(args)
				if err != nil {
					return fmt.Errorf("failed to expand patterns %v: %w", args, err)
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
					pkg, err := xctx.Import(pkgPath, currentDirectory, 0)
					if s.Watcher != nil && pkg != nil { // add watch even on error
						s.Watcher.Add(pkg.Dir)
					}
					if err != nil {
						return err
					}
					archive, err := s.BuildProject(pkg)
					if err != nil {
						return err
					}

					if pkg.IsCommand() && !pkg.UpToDate {
						if err := s.WriteCommandPackage(archive, pkg.InstallPath()); err != nil {
							return err
						}
					}
				}
				return nil
			}()

			if s.Watcher == nil {
				return err
			} else if err != nil {
				handleError(err, options, nil)
			}
			s.WaitForChange()
		}
	}

	cmdDoc := &cobra.Command{
		Use:   "doc [arguments]",
		Short: "display documentation for the requested, package, method or symbol",
	}
	cmdDoc.RunE = func(cmd *cobra.Command, args []string) error {
		goDoc := exec.Command("go", append([]string{"doc"}, args...)...)
		goDoc.Stdout = os.Stdout
		goDoc.Stderr = os.Stderr
		goDoc.Env = append(os.Environ(), "GOARCH=js")
		return goDoc.Run()
	}

	cmdGet := &cobra.Command{
		Use:   "get [packages]",
		Short: "download and install packages and dependencies",
	}
	cmdGet.Flags().AddFlagSet(flagVerbose)
	cmdGet.Flags().AddFlagSet(flagQuiet)
	cmdGet.Flags().AddFlagSet(compilerFlags)
	cmdGet.Run = cmdInstall.Run

	cmdRun := &cobra.Command{
		Use:   "run [gofiles...] [arguments...]",
		Short: "compile and run Go program",
	}
	cmdRun.Flags().AddFlagSet(flagVerbose)
	cmdRun.Flags().AddFlagSet(flagQuiet)
	cmdRun.Flags().AddFlagSet(compilerFlags)
	cmdRun.RunE = func(cmd *cobra.Command, args []string) error {
		options.BuildTags = strings.Fields(tags)
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

		tempfile, err := os.CreateTemp(currentDirectory, filepath.Base(args[0])+".")
		if err != nil && strings.HasPrefix(currentDirectory, runtime.GOROOT()) {
			tempfile, err = os.CreateTemp("", filepath.Base(args[0])+".")
		}
		if err != nil {
			return err
		}
		defer func() {
			tempfile.Close()
			os.Remove(tempfile.Name())
			os.Remove(tempfile.Name() + ".map")
		}()
		s, err := gbuild.NewSession(options)
		if err != nil {
			return err
		}
		if err := s.BuildFiles(args[:lastSourceArg], tempfile.Name(), currentDirectory); err != nil {
			return err
		}
		if err := runNode(tempfile.Name(), args[lastSourceArg:], "", options.Quiet, nil); err != nil {
			return err
		}
		return nil
	}

	cmdTest := &cobra.Command{
		Use:   "test [packages]",
		Short: "test packages",
	}
	bench := cmdTest.Flags().String("bench", "", "Run benchmarks matching the regular expression. By default, no benchmarks run. To run all benchmarks, use '--bench=.'.")
	benchtime := cmdTest.Flags().String("benchtime", "", "Run enough iterations of each benchmark to take t, specified as a time.Duration (for example, -benchtime 1h30s). The default is 1 second (1s).")
	count := cmdTest.Flags().String("count", "", "Run each test and benchmark n times (default 1). Examples are always run once.")
	run := cmdTest.Flags().String("run", "", "Run only those tests and examples matching the regular expression.")
	short := cmdTest.Flags().Bool("short", false, "Tell long-running tests to shorten their run time.")
	verbose := cmdTest.Flags().BoolP("verbose", "v", false, "Log all tests as they are run. Also print all text from Log and Logf calls even if the test succeeds.")
	compileOnly := cmdTest.Flags().BoolP("compileonly", "c", false, "Compile the test binary to pkg.test.js but do not run it (where pkg is the last element of the package's import path). The file name can be changed with the -o flag.")
	outputFilename := cmdTest.Flags().StringP("output", "o", "", "Compile the test binary to the named file. The test still runs (unless -c is specified).")
	parallelTests := cmdTest.Flags().IntP("parallel", "p", runtime.NumCPU(), "Allow running tests in parallel for up to -p packages. Tests within the same package are still executed sequentially.")
	cmdTest.Flags().AddFlagSet(compilerFlags)
	cmdTest.RunE = func(cmd *cobra.Command, args []string) error {
		options.BuildTags = strings.Fields(tags)

		// Expand import path patterns.
		patternContext := gbuild.NewBuildContext("", options.BuildTags)
		matches, err := patternContext.Match(args)
		if err != nil {
			return fmt.Errorf("failed to expand patterns %v: %w", args, err)
		}

		if *compileOnly && len(matches) > 1 {
			return errors.New("cannot use -c flag with multiple packages")
		}
		if *outputFilename != "" && len(matches) > 1 {
			return errors.New("cannot use -o flag with multiple packages")
		}
		if *parallelTests < 1 {
			return errors.New("--parallel cannot be less than 1")
		}

		parallelSlots := make(chan (bool), *parallelTests) // Semaphore for parallel test executions.
		if len(matches) == 1 {
			// Disable output buffering if testing only one package.
			parallelSlots = make(chan (bool), 1)
		}
		executions := errgroup.Group{}

		pkgs := make([]*gbuild.PackageData, len(matches))
		for i, pkgPath := range matches {
			var err error
			pkgs[i], err = gbuild.Import(pkgPath, 0, "", options.BuildTags)
			if err != nil {
				return err
			}
		}

		var (
			exitErr   error
			exitErrMu = &sync.Mutex{}
		)
		for _, pkg := range pkgs {
			pkg := pkg // Capture for the goroutine.
			if len(pkg.TestGoFiles) == 0 && len(pkg.XTestGoFiles) == 0 {
				fmt.Printf("?   \t%s\t[no test files]\n", pkg.ImportPath)
				continue
			}
			localOpts := options
			localOpts.TestedPackage = pkg.ImportPath
			s, err := gbuild.NewSession(localOpts)
			if err != nil {
				return err
			}

			pkg.IsTest = true
			mainPkgArchive, err := s.BuildProject(pkg)
			if err != nil {
				return fmt.Errorf("failed to compile testmain package for %s: %w", pkg.ImportPath, err)
			}

			if *compileOnly && *outputFilename == "" {
				*outputFilename = pkg.Package.Name + "_test.js"
			}

			var outfile *os.File
			if *outputFilename != "" {
				outfile, err = os.Create(*outputFilename)
				if err != nil {
					return err
				}
			} else {
				outfile, err = os.CreateTemp(currentDirectory, pkg.Package.Name+"_test.*.js")
				if err != nil {
					return err
				}
				outfile.Close() // Release file handle early, we only need the name.
			}
			cleanupTemp := func() {
				if *outputFilename == "" {
					os.Remove(outfile.Name())
					os.Remove(outfile.Name() + ".map")
				}
			}
			defer cleanupTemp() // Safety net in case cleanup after execution doesn't happen.

			if err := s.WriteCommandPackage(mainPkgArchive, outfile.Name()); err != nil {
				return err
			}

			if *compileOnly {
				continue
			}

			var args []string
			if *bench != "" {
				args = append(args, "-test.bench", *bench)
			}
			if *benchtime != "" {
				args = append(args, "-test.benchtime", *benchtime)
			}
			if *count != "" {
				args = append(args, "-test.count", *count)
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
			executions.Go(func() error {
				parallelSlots <- true              // Acquire slot
				defer func() { <-parallelSlots }() // Release slot

				status := "ok  "
				start := time.Now()
				var testOut io.ReadWriter
				if cap(parallelSlots) > 1 {
					// If running in parallel, capture test output in a temporary buffer to avoid mixing
					// output from different tests and print it later.
					testOut = &bytes.Buffer{}
				}

				err := runNode(outfile.Name(), args, runTestDir(pkg), options.Quiet, testOut)

				cleanupTemp() // Eagerly cleanup temporary compiled files after execution.

				if testOut != nil {
					io.Copy(os.Stdout, testOut)
				}

				if err != nil {
					if _, ok := err.(*exec.ExitError); !ok {
						return err
					}
					exitErrMu.Lock()
					exitErr = err
					exitErrMu.Unlock()
					status = "FAIL"
				}
				fmt.Printf("%s\t%s\t%.3fs\n", status, pkg.ImportPath, time.Since(start).Seconds())
				return nil
			})
		}
		if err := executions.Wait(); err != nil {
			return err
		}
		return exitErr
	}

	cmdServe := &cobra.Command{
		Use:   "serve [root]",
		Short: "compile on-the-fly and serve",
	}
	cmdServe.Args = cobra.MaximumNArgs(1)
	cmdServe.Flags().AddFlagSet(flagVerbose)
	cmdServe.Flags().AddFlagSet(flagQuiet)
	cmdServe.Flags().AddFlagSet(compilerFlags)
	var addr string
	cmdServe.Flags().StringVarP(&addr, "http", "", ":8080", "HTTP bind address to serve")
	cmdServe.RunE = func(cmd *cobra.Command, args []string) error {
		options.BuildTags = strings.Fields(tags)
		var root string

		if len(args) == 1 {
			root = args[0]
		}

		// Create a new session eagerly to check if it fails, and report the error right away.
		// Otherwise, users will see it only after trying to serve a package, which is a bad experience.
		_, err := gbuild.NewSession(options)
		if err != nil {
			return err
		}
		sourceFiles := http.FileServer(serveCommandFileSystem{
			serveRoot:  root,
			options:    options,
			sourceMaps: make(map[string][]byte),
		})

		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		if tcpAddr := ln.Addr().(*net.TCPAddr); tcpAddr.IP.Equal(net.IPv4zero) || tcpAddr.IP.Equal(net.IPv6zero) { // Any available addresses.
			fmt.Printf("serving at http://localhost:%d and on port %d of any available addresses\n", tcpAddr.Port, tcpAddr.Port)
		} else { // Specific address.
			fmt.Printf("serving at http://%s\n", tcpAddr)
		}
		fmt.Fprintln(os.Stderr, http.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)}, sourceFiles))
		return nil
	}

	cmdVersion := &cobra.Command{
		Use:   "version",
		Short: "print GopherJS compiler version",
		Args:  cobra.ExactArgs(0),
	}
	cmdVersion.Run = func(cmd *cobra.Command, args []string) {
		fmt.Printf("GopherJS %s\n", compiler.Version)
	}

	cmdClean := &cobra.Command{
		Use:   "clean",
		Short: "clean GopherJS build cache",
	}
	cmdClean.RunE = func(cmd *cobra.Command, args []string) error {
		return cache.Clear()
	}

	rootCmd := &cobra.Command{
		Use:           "gopherjs",
		Long:          "GopherJS is a tool for compiling Go source code to JavaScript.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	rootCmd.AddCommand(cmdBuild, cmdGet, cmdInstall, cmdRun, cmdTest, cmdServe, cmdVersion, cmdDoc, cmdClean)

	{
		var logLevel string
		var cpuProfile string
		var allocProfile string
		rootCmd.PersistentFlags().StringVar(&logLevel, "log_level", log.ErrorLevel.String(), "Compiler log level (debug, info, warn, error, fatal, panic).")
		rootCmd.PersistentFlags().StringVar(&cpuProfile, "cpu_profile", "", "Save GopherJS compiler CPU profile at the given path. If unset, profiling is disabled.")
		rootCmd.PersistentFlags().StringVar(&allocProfile, "alloc_profile", "", "Save GopherJS compiler allocation profile at the given path. If unset, profiling is disabled.")

		rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
			lvl, err := log.ParseLevel(logLevel)
			if err != nil {
				return fmt.Errorf("invalid --log_level value %q: %w", logLevel, err)
			}
			log.SetLevel(lvl)

			if cpuProfile != "" {
				f, err := os.Create(cpuProfile)
				if err != nil {
					return fmt.Errorf("failed to create CPU profile file at %q: %w", cpuProfile, err)
				}
				if err := pprof.StartCPUProfile(f); err != nil {
					return fmt.Errorf("failed to start CPU profile: %w", err)
				}
				// Not closing the file here, since we'll be writing to it throughout
				// the lifetime of the process. It will be closed automatically when
				// the process terminates.
			}
			return nil
		}
		rootCmd.PersistentPostRunE = func(cmd *cobra.Command, args []string) error {
			if cpuProfile != "" {
				pprof.StopCPUProfile()
			}
			if allocProfile != "" {
				f, err := os.Create(allocProfile)
				if err != nil {
					return fmt.Errorf("failed to create alloc profile file at %q: %w", allocProfile, err)
				}
				if err := pprof.Lookup("allocs").WriteTo(f, 0); err != nil {
					return fmt.Errorf("failed to write alloc profile: %w", err)
				}
				f.Close()
			}
			return nil
		}
	}
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(handleError(err, options, nil))
	}
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
	sourceMaps map[string][]byte
}

func (fs serveCommandFileSystem) Open(requestName string) (http.File, error) {
	name := path.Join(fs.serveRoot, requestName[1:]) // requestName[0] == '/'
	log.Printf("Request: %s", name)

	dir, file := path.Split(name)
	base := path.Base(dir) // base is parent folder name, which becomes the output file name.

	isPkg := file == base+".js"
	isMap := file == base+".js.map"
	isIndex := file == "index.html"

	// Create a new session to pick up changes to source code on disk.
	// TODO(dmitshur): might be possible to get a single session to detect changes to source code on disk
	s, err := gbuild.NewSession(fs.options)
	if err != nil {
		return nil, err
	}

	if isPkg || isMap || isIndex {
		// If we're going to be serving our special files, make sure there's a Go command in this folder.
		pkg, err := gbuild.Import(path.Dir(name), 0, s.InstallSuffix(), fs.options.BuildTags)
		if err != nil || pkg.Name != "main" {
			isPkg = false
			isMap = false
			isIndex = false
		}

		switch {
		case isPkg:
			buf := new(bytes.Buffer)
			browserErrors := new(bytes.Buffer)
			err := func() error {
				archive, err := s.BuildProject(pkg)
				if err != nil {
					return err
				}

				sourceMapFilter := &compiler.SourceMapFilter{Writer: buf}
				m := &sourcemap.Map{File: base + ".js"}
				sourceMapFilter.MappingCallback = s.SourceMappingCallback(m)

				deps, err := compiler.ImportDependencies(archive, s.ImportResolverFor(""))
				if err != nil {
					return err
				}
				if err := compiler.WriteProgramCode(deps, sourceMapFilter, s.GoRelease()); err != nil {
					return err
				}

				mapBuf := new(bytes.Buffer)
				m.WriteTo(mapBuf)
				buf.WriteString("//# sourceMappingURL=" + base + ".js.map\n")
				fs.sourceMaps[name+".map"] = mapBuf.Bytes()

				return nil
			}()
			handleError(err, fs.options, browserErrors)
			if err != nil {
				buf = browserErrors
			}
			return newFakeFile(base+".js", buf.Bytes()), nil

		case isMap:
			if content, ok := fs.sourceMaps[name]; ok {
				return newFakeFile(base+".js.map", content), nil
			}
		}
	}

	// First try to serve the request with a root prefix supplied in the CLI.
	if f, err := fs.serveSourceTree(s.XContext(), name); err == nil {
		return f, nil
	}

	// If that didn't work, try without the prefix.
	if f, err := fs.serveSourceTree(s.XContext(), requestName); err == nil {
		return f, nil
	}

	if isIndex {
		// If there was no index.html file in any dirs, supply our own.
		return newFakeFile("index.html", []byte(`<html><head><meta charset="utf-8"><script src="`+base+`.js"></script></head><body></body></html>`)), nil
	}

	return nil, os.ErrNotExist
}

func (fs serveCommandFileSystem) serveSourceTree(xctx gbuild.XContext, reqPath string) (http.File, error) {
	parts := strings.Split(path.Clean(reqPath), "/")
	// Under Go Modules different packages can be located in different module
	// directories, which no longer align with import paths.
	//
	// We don't know which part of the requested path is package import path and
	// which is a path under the package directory, so we try different split
	// points until the package is found successfully.
	for i := len(parts); i > 0; i-- {
		pkgPath := path.Clean(path.Join(parts[:i]...))
		filePath := path.Clean(path.Join(parts[i:]...))
		if pkg, err := xctx.Import(pkgPath, ".", build.FindOnly); err == nil {
			return http.Dir(pkg.Dir).Open(filePath)
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

// handleError handles err and returns an appropriate exit code.
// If browserErrors is non-nil, errors are written for presentation in browser.
func handleError(err error, options *gbuild.Options, browserErrors *bytes.Buffer) int {
	switch err := err.(type) {
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

// runNode runs script with args using Node.js in directory dir.
// If dir is empty string, current directory is used.
// Is out is not nil, process stderr and stdout are redirected to it, otherwise
// os.Stdout and os.Stderr are used.
func runNode(script string, args []string, dir string, quiet bool, out io.Writer) error {
	var allArgs []string
	if b, _ := strconv.ParseBool(os.Getenv("SOURCE_MAP_SUPPORT")); os.Getenv("SOURCE_MAP_SUPPORT") == "" || b {
		allArgs = []string{"--enable-source-maps"}
	}

	if runtime.GOOS != "windows" {
		// We've seen issues with stack space limits causing
		// recursion-heavy standard library tests to fail (e.g., see
		// https://github.com/gopherjs/gopherjs/pull/669#issuecomment-319319483).
		//
		// There are two separate limits in non-Windows environments:
		//
		// -	OS process limit
		// -	Node.js (V8) limit
		//
		// GopherJS fetches the current OS process limit, and sets the Node.js limit
		// to a value slightly below it (otherwise nodejs is likely to segfault).
		// The backoff size has been determined experimentally on a linux machine,
		// so it may not be 100% reliable. So both limits are kept in sync and can
		// be controlled by setting OS process limit. E.g.:
		//
		// 	ulimit -s 10000 && gopherjs test
		//
		cur, err := sysutil.RlimitStack()
		if err != nil {
			return fmt.Errorf("failed to get stack size limit: %v", err)
		}
		cur = cur / 1024           // Convert bytes to KiB.
		defaultSize := uint64(984) // --stack-size default value.
		if backoff := uint64(64); cur > defaultSize+backoff {
			cur = cur - backoff
		}
		allArgs = append(allArgs, fmt.Sprintf("--stack_size=%v", cur))
	}

	allArgs = append(allArgs, script)
	allArgs = append(allArgs, args...)

	node := exec.Command("node", allArgs...)
	node.Dir = dir
	node.Stdin = os.Stdin
	if out != nil {
		node.Stdout = out
		node.Stderr = out
	} else {
		node.Stdout = os.Stdout
		node.Stderr = os.Stderr
	}
	err := node.Run()
	if _, ok := err.(*exec.ExitError); err != nil && !ok {
		err = fmt.Errorf("could not run Node.js: %s", err.Error())
	}
	return err
}

// runTestDir returns the directory for Node.js to use when running tests for package p.
// Empty string means current directory.
func runTestDir(p *gbuild.PackageData) string {
	if p.IsVirtual {
		// The package is virtual and doesn't have a physical directory. Use current directory.
		return ""
	}
	// Run tests in the package directory.
	return p.Dir
}
