package main

import (
	"bitbucket.org/kardianos/osext"
	"code.google.com/p/go.exp/fsnotify"
	"code.google.com/p/go.tools/go/types"
	"flag"
	"fmt"
	"github.com/gopherjs/gopherjs/compiler"
	"github.com/neelance/sourcemap"
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
	Archive    *compiler.Archive
}

type importCError struct{}

func (e *importCError) Error() string {
	return `importing "C" is not supported by GopherJS`
}

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

type session struct {
	t        *compiler.Compiler
	packages map[string]*packageData
	verbose  bool
	watcher  *fsnotify.Watcher
}

func NewSession(verbose bool, watch bool) *session {
	s := &session{
		t:        compiler.New(),
		verbose:  verbose || watch,
		packages: make(map[string]*packageData),
	}
	if watch {
		var err error
		s.watcher, err = fsnotify.NewWatcher()
		if err != nil {
			panic(err)
		}
	}
	return s
}

func main() {
	flag.Parse()

	cmd := flag.Arg(0)
	switch cmd {
	case "build":
		buildFlags := flag.NewFlagSet("build", flag.ContinueOnError)
		var pkgObj string
		buildFlags.StringVar(&pkgObj, "o", "", "")
		verbose := buildFlags.Bool("v", false, "print the names of packages as they are compiled")
		watch := buildFlags.Bool("w", false, "watch for changes to the source files")
		buildFlags.Parse(flag.Args()[1:])

		for {
			s := NewSession(*verbose, *watch)

			exitCode := handleError(func() error {
				if buildFlags.NArg() == 0 {
					buildContext := &build.Context{
						GOROOT:   build.Default.GOROOT,
						GOPATH:   build.Default.GOPATH,
						GOOS:     build.Default.GOOS,
						GOARCH:   "js",
						Compiler: "gc",
					}
					if s.watcher != nil {
						s.watcher.Watch(currentDirectory)
					}
					buildPkg, err := buildContext.ImportDir(currentDirectory, 0)
					if err != nil {
						return err
					}
					pkg := &packageData{Package: buildPkg}
					pkg.ImportPath = currentDirectory
					if err := s.buildPackage(pkg); err != nil {
						return err
					}
					if pkgObj == "" {
						pkgObj = filepath.Base(currentDirectory) + ".js"
					}
					if err := s.writeCommandPackage(pkg, pkgObj); err != nil {
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
					names := make([]string, buildFlags.NArg())
					for i, name := range buildFlags.Args() {
						name = filepath.ToSlash(name)
						names[i] = name
						if s.watcher != nil {
							s.watcher.Watch(filepath.ToSlash(name))
						}
					}
					if err := s.buildFiles(buildFlags.Args(), pkgObj); err != nil {
						return err
					}
					return nil
				}

				for _, pkgPath := range buildFlags.Args() {
					pkgPath = filepath.ToSlash(pkgPath)
					if s.watcher != nil {
						s.watcher.Watch(pkgPath)
					}
					buildPkg, err := buildImport(pkgPath, 0)
					if err != nil {
						return err
					}
					pkg := &packageData{Package: buildPkg}
					if err := s.buildPackage(pkg); err != nil {
						return err
					}
					if pkgObj == "" {
						pkgObj = filepath.Base(buildFlags.Arg(0)) + ".js"
					}
					if err := s.writeCommandPackage(pkg, pkgObj); err != nil {
						return err
					}
				}
				return nil
			})

			if s.watcher == nil {
				os.Exit(exitCode)
			}
			s.waitForChange()
		}

	case "install":
		installFlags := flag.NewFlagSet("install", flag.ContinueOnError)
		verbose := installFlags.Bool("v", false, "print the names of packages as they are compiled")
		watch := installFlags.Bool("w", false, "watch for changes to the source files")
		installFlags.Parse(flag.Args()[1:])

		for {
			s := NewSession(*verbose, *watch)

			exitCode := handleError(func() error {
				pkgs := installFlags.Args()
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
					pkgPath = filepath.ToSlash(pkgPath)
					if _, err := s.importPackage(pkgPath); err != nil {
						return err
					}
					pkg := s.packages[pkgPath]
					if err := s.writeCommandPackage(pkg, pkg.PkgObj); err != nil {
						return err
					}
				}
				return nil
			})

			if s.watcher == nil {
				os.Exit(exitCode)
			}
			s.waitForChange()
		}

	case "run":
		os.Exit(handleError(func() error {
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

			s := NewSession(false, false)
			if err := s.buildFiles(flag.Args()[1:lastSourceArg], tempfile.Name()); err != nil {
				return err
			}
			if err := runNode(tempfile.Name(), flag.Args()[lastSourceArg:], ""); err != nil {
				return err
			}
			return nil
		}))

	case "test":
		testFlags := flag.NewFlagSet("test", flag.ContinueOnError)
		verbose := testFlags.Bool("v", false, "verbose")
		short := testFlags.Bool("short", false, "short")
		testFlags.Parse(flag.Args()[1:])

		os.Exit(handleError(func() error {
			pkgs := make([]*build.Package, testFlags.NArg())
			for i, pkgPath := range testFlags.Args() {
				pkgPath = filepath.ToSlash(pkgPath)
				var err error
				pkgs[i], err = buildImport(pkgPath, 0)
				if err != nil {
					return err
				}
			}
			if len(pkgs) == 0 {
				srcDir, err := filepath.EvalSymlinks(filepath.Join(build.Default.GOPATH, "src"))
				if err != nil {
					return err
				}
				var pkg *build.Package
				if strings.HasPrefix(currentDirectory, srcDir) {
					pkgPath, err := filepath.Rel(srcDir, currentDirectory)
					if err != nil {
						return err
					}
					if pkg, err = buildImport(pkgPath, 0); err != nil {
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
				pkg := &packageData{Package: buildPkg}
				s := NewSession(false, false)
				if err := s.buildPackage(pkg); err != nil {
					return err
				}

				mainPkg := &packageData{
					Package: &build.Package{
						Name:       "main",
						ImportPath: "main",
					},
					Archive: &compiler.Archive{
						ImportPath: "main",
					},
				}
				s.packages["main"] = mainPkg
				s.t.NewEmptyTypesPackage("main")
				testingOutput, err := s.importPackage("testing")
				if err != nil {
					panic(err)
				}
				mainPkg.Archive.AddDependenciesOf(testingOutput)

				var mainFunc compiler.Decl
				var names []string
				var tests []string
				collectTests := func(pkg *packageData) {
					for _, name := range pkg.Archive.Tests {
						names = append(names, name)
						tests = append(tests, fmt.Sprintf(`go$packages["%s"].%s`, pkg.ImportPath, name))
						mainFunc.DceDeps = append(mainFunc.DceDeps, compiler.DepId(pkg.ImportPath+":"+name))
					}
					mainPkg.Archive.AddDependenciesOf(pkg.Archive)
				}

				collectTests(pkg)
				if len(pkg.XTestGoFiles) != 0 {
					testPkg := &packageData{Package: &build.Package{
						ImportPath: pkg.ImportPath + "_test",
						Dir:        pkg.Dir,
						GoFiles:    pkg.XTestGoFiles,
					}}
					if err := s.buildPackage(testPkg); err != nil {
						return err
					}
					collectTests(testPkg)
				}

				mainFunc.DceDeps = append(mainFunc.DceDeps, compiler.DepId("flag:Parse"))
				mainFunc.BodyCode = []byte(fmt.Sprintf(`
				go$pkg.main = function() {
					var testing = go$packages["testing"];
					testing.Main2("%s", "%s", new (go$sliceType(Go$String))(["%s"]), new (go$sliceType(go$funcType([testing.T.Ptr], [], false)))([%s]));
				};
			`, pkg.ImportPath, pkg.Dir, strings.Join(names, `", "`), strings.Join(tests, ", ")))

				mainPkg.Archive.Declarations = []compiler.Decl{mainFunc}
				mainPkg.Archive.AddDependency("main")

				tempfile, err := ioutil.TempFile("", "test.")
				if err != nil {
					return err
				}
				defer func() {
					tempfile.Close()
					os.Remove(tempfile.Name())
				}()

				if err := s.writeCommandPackage(mainPkg, tempfile.Name()); err != nil {
					return err
				}

				var args []string
				if *verbose {
					args = append(args, "-test.v")
				}
				if *short {
					args = append(args, "-test.short")
				}
				if err := runNode(tempfile.Name(), args, ""); err != nil {
					if _, ok := err.(*exec.ExitError); !ok {
						return err
					}
					exitErr = err
				}
			}
			return exitErr
		}))

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

		os.Exit(handleError(func() error {
			if len(tool) == 2 {
				switch tool[1] {
				case 'g':
					basename := filepath.Base(toolFlags.Arg(0))
					s := NewSession(false, false)
					if err := s.buildFiles([]string{toolFlags.Arg(0)}, basename[:len(basename)-3]+".js"); err != nil {
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

func (s *session) buildFiles(filenames []string, pkgObj string) error {
	pkg := &packageData{
		Package: &build.Package{
			Name:       "main",
			ImportPath: "main",
			Dir:        currentDirectory,
			GoFiles:    filenames,
		},
	}

	if err := s.buildPackage(pkg); err != nil {
		return err
	}
	return s.writeCommandPackage(pkg, pkgObj)
}

func (s *session) importPackage(path string) (*compiler.Archive, error) {
	if pkg, found := s.packages[path]; found {
		return pkg.Archive, nil
	}

	buildPkg, err := buildImport(path, build.AllowBinary)
	if s.watcher != nil && buildPkg != nil { // add watch even on error
		if err := s.watcher.Watch(buildPkg.Dir); err != nil {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}
	pkg := &packageData{Package: buildPkg}
	if err := s.buildPackage(pkg); err != nil {
		return nil, err
	}
	return pkg.Archive, nil
}

func (s *session) buildPackage(pkg *packageData) error {
	s.packages[pkg.ImportPath] = pkg
	if pkg.ImportPath == "unsafe" {
		return nil
	}

	if pkg.PkgObj != "" {
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
			_, err := s.importPackage(importedPkgPath)
			if err != nil {
				return err
			}
			impModeTime := s.packages[importedPkgPath].SrcModTime
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

			pkg.Archive, err = s.t.UnmarshalArchive(pkg.PkgObj, pkg.ImportPath, objFile)
			if err != nil {
				return err
			}

			return nil
		}
	}

	fileSet := token.NewFileSet()
	var files []*ast.File
	var errList compiler.ErrorList

	replacedDeclNames := make(map[string]bool)
	funcName := func(d *ast.FuncDecl) string {
		if d.Recv == nil {
			return d.Name.Name
		}
		recv := d.Recv.List[0].Type
		if star, ok := recv.(*ast.StarExpr); ok {
			recv = star.X
		}
		return recv.(*ast.Ident).Name + "." + d.Name.Name
	}
	if nativesPkg, err := buildImport("github.com/gopherjs/gopherjs/compiler/natives/"+pkg.ImportPath, 0); err == nil {
		for _, name := range nativesPkg.GoFiles {
			file, err := parser.ParseFile(fileSet, filepath.Join(nativesPkg.Dir, name), nil, 0)
			if err != nil {
				panic(err)
			}
			for _, decl := range file.Decls {
				if d, ok := decl.(*ast.FuncDecl); ok {
					replacedDeclNames[funcName(d)] = true
				}
			}
			files = append(files, file)
		}
	}
	delete(replacedDeclNames, "init")

	for _, name := range pkg.GoFiles {
		if !filepath.IsAbs(name) {
			name = filepath.Join(pkg.Dir, name)
		}
		r, err := os.Open(name)
		if err != nil {
			return err
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

		for _, decl := range file.Decls {
			if d, ok := decl.(*ast.FuncDecl); ok && replacedDeclNames[funcName(d)] {
				d.Name = ast.NewIdent("_")
			}
		}

		files = append(files, file)
	}
	if errList != nil {
		return errList
	}

	var err error
	pkg.Archive, err = s.t.Compile(pkg.ImportPath, files, fileSet, s.importPackage)
	if err != nil {
		return err
	}

	if s.verbose {
		fmt.Println(pkg.ImportPath)
	}

	if pkg.PkgObj == "" || pkg.IsCommand() {
		return nil
	}

	if err := s.writeLibraryPackage(pkg, pkg.PkgObj); err != nil {
		if strings.HasPrefix(pkg.PkgObj, build.Default.GOROOT) {
			// fall back to GOPATH
			if err := s.writeLibraryPackage(pkg, build.Default.GOPATH+pkg.PkgObj[len(build.Default.GOROOT):]); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	return nil
}

func (s *session) writeLibraryPackage(pkg *packageData, pkgObj string) error {
	if err := os.MkdirAll(filepath.Dir(pkgObj), 0777); err != nil {
		return err
	}

	data, err := s.t.MarshalArchive(pkg.Archive)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(pkgObj, data, 0666)
}

func (s *session) writeCommandPackage(pkg *packageData, pkgObj string) error {
	if !pkg.IsCommand() || pkg.UpToDate {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(pkgObj), 0777); err != nil {
		return err
	}
	codeFile, err := os.Create(pkgObj)
	if err != nil {
		return err
	}
	defer codeFile.Close()
	mapFile, err := os.Create(pkgObj + ".map")
	if err != nil {
		return err
	}
	defer mapFile.Close()

	var allPkgs []*compiler.Archive
	for _, depPath := range pkg.Archive.Dependencies {
		dep, err := s.importPackage(depPath)
		if err != nil {
			return err
		}
		allPkgs = append(allPkgs, dep)
	}

	m := sourcemap.Map{File: filepath.Base(pkgObj)}
	s.t.WriteProgramCode(allPkgs, pkg.ImportPath, &compiler.SourceMapFilter{Writer: codeFile, MappingCallback: func(generatedLine, generatedColumn int, fileSet *token.FileSet, originalPos token.Pos) {
		if !originalPos.IsValid() {
			m.AddMapping(&sourcemap.Mapping{GeneratedLine: generatedLine, GeneratedColumn: generatedColumn})
			return
		}
		pos := fileSet.Position(originalPos)
		file := pos.Filename
		switch {
		case strings.HasPrefix(file, build.Default.GOPATH):
			file = filepath.ToSlash(filepath.Join("/gopath", file[len(build.Default.GOPATH):]))
		case strings.HasPrefix(file, build.Default.GOROOT):
			file = filepath.ToSlash(filepath.Join("/goroot", file[len(build.Default.GOROOT):]))
		default:
			file = filepath.Base(file)
		}
		m.AddMapping(&sourcemap.Mapping{GeneratedLine: generatedLine, GeneratedColumn: generatedColumn, OriginalFile: file, OriginalLine: pos.Line, OriginalColumn: pos.Column})
	}})
	fmt.Fprintf(codeFile, "//# sourceMappingURL=%s.map\n", filepath.Base(pkgObj))
	m.WriteTo(mapFile)

	return nil
}

func (s *session) waitForChange() {
	fmt.Println("\x1B[32mwatching for changes...\x1B[39m")
	select {
	case ev := <-s.watcher.Event:
		fmt.Println("\x1B[32mchange detected: " + ev.Name + "\x1B[39m")
	case err := <-s.watcher.Error:
		fmt.Println("\x1B[32mwatcher error: " + err.Error() + "\x1B[39m")
	}
	s.watcher.Close()
}

func runNode(script string, args []string, dir string) error {
	node := exec.Command("node", append([]string{script}, args...)...)
	node.Dir = dir
	node.Stdin = os.Stdin
	node.Stdout = os.Stdout
	node.Stderr = os.Stderr
	if err := node.Run(); err != nil {
		return fmt.Errorf("could not run Node.js: %s", err.Error())
	}
	return nil
}
