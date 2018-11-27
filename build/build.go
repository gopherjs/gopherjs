package build

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/gopherjs/gopherjs/compiler"
	"github.com/gopherjs/gopherjs/compiler/gopherjspkg"
	"github.com/gopherjs/gopherjs/compiler/natives"
	"github.com/neelance/sourcemap"
	"github.com/rogpeppe/go-internal/cache"
	"github.com/shurcooL/httpfs/vfsutil"
	"golang.org/x/tools/go/buildutil"
)

const (
	hashDebug = false

	buildCacheDirName = "gopherjs-build"
)

var (
	compilerBinaryHash string
)

func init() {
	// We do this here because it will only fail in truly bad situations, i.e.
	// machine running out of resources. We also panic if there is a problem
	// because it's unlikely anything else will be useful/work
	h, err := hashCompilerBinary()
	if err != nil {
		panic(err)
	}
	compilerBinaryHash = h
}

type ImportCError struct {
	pkgPath string
}

func (e *ImportCError) Error() string {
	return e.pkgPath + `: importing "C" is not supported by GopherJS`
}

// NewBuildContext creates a build context for building Go packages
// with GopherJS compiler.
//
// Core GopherJS packages (i.e., "github.com/gopherjs/gopherjs/js", "github.com/gopherjs/gopherjs/nosync")
// are loaded from gopherjspkg.FS virtual filesystem rather than GOPATH.
func NewBuildContext(installSuffix string, buildTags []string) *build.Context {
	gopherjsRoot := filepath.Join(build.Default.GOROOT, "src", "github.com", "gopherjs", "gopherjs")

	ctxt := build.Default
	ctxt.GOARCH = "js"
	ctxt.Compiler = "gc"
	ctxt.BuildTags = append(buildTags,
		"netgo",            // See https://godoc.org/net#hdr-Name_Resolution.
		"purego",           // See https://golang.org/issues/23172.
		"js",               // this effectively identifies that we are GopherJS
		"!wasm",            // but not webassembly
		"math_big_pure_go", // Use pure Go version of math/big; we don't want non-Go assembly versions.
	)

	// TODO this is not great; the build use by GopherJS should not
	// be a function of the package imported. See below for check

	ctxt.CgoEnabled = true // detect `import "C"` to throw proper error

	foundGopherJSDev := false
	for _, t := range buildTags {
		if t == "gopherjsdev" {
			foundGopherJSDev = true
		}
	}

	if !foundGopherJSDev {
		ctxt.IsDir = func(path string) bool {
			if strings.HasPrefix(path, gopherjsRoot+string(filepath.Separator)) {
				path = filepath.ToSlash(path[len(gopherjsRoot):])
				if fi, err := vfsutil.Stat(gopherjspkg.FS, path); err == nil {
					return fi.IsDir()
				}
			}
			fi, err := os.Stat(path)
			return err == nil && fi.IsDir()
		}
		ctxt.ReadDir = func(path string) ([]os.FileInfo, error) {
			if strings.HasPrefix(path, gopherjsRoot+string(filepath.Separator)) {
				path = filepath.ToSlash(path[len(gopherjsRoot):])
				if fis, err := vfsutil.ReadDir(gopherjspkg.FS, path); err == nil {
					return fis, nil
				}
			}
			return ioutil.ReadDir(path)
		}
		ctxt.OpenFile = func(path string) (io.ReadCloser, error) {
			if strings.HasPrefix(path, gopherjsRoot+string(filepath.Separator)) {
				path = filepath.ToSlash(path[len(gopherjsRoot):])
				if f, err := gopherjspkg.FS.Open(path); err == nil {
					return f, nil
				}
			}
			return os.Open(path)
		}
	}

	return &ctxt
}

// Import returns details about the Go package named by the import path. If the
// path is a local import path naming a package that can be imported using
// a standard import path, the returned package will set p.ImportPath to
// that path.
//
// In the directory containing the package, .go and .inc.js files are
// considered part of the package except for:
//
//    - .go files in package documentation
//    - files starting with _ or . (likely editor temporary files)
//    - files with build constraints not satisfied by the context
//
// If an error occurs, Import returns a non-nil error and a nil
// *PackageData.
func (s *Session) Import(path string, mode build.ImportMode, installSuffix string, buildTags []string) (*PackageData, error) {
	wd, err := os.Getwd()
	if err != nil {
		// Getwd may fail if we're in GOARCH=js mode. That's okay, handle
		// it by falling back to empty working directory. It just means
		// Import will not be able to resolve relative import paths.
		wd = "."
	}
	bctx := NewBuildContext(installSuffix, buildTags)
	return s.importWithSrcDir(*bctx, path, wd, mode, installSuffix)
}

func (s *Session) importWithSrcDir(bctx build.Context, path string, srcDir string, mode build.ImportMode, installSuffix string) (*PackageData, error) {
	// bctx is passed by value, so it can be modified here.
	var isVirtual bool
	switch path {
	case "syscall":
		// syscall needs to use a typical GOARCH like amd64 to pick up definitions for _Socklen, BpfInsn, IFNAMSIZ, Timeval, BpfStat, SYS_FCNTL, Flock_t, etc.
		bctx.GOARCH = runtime.GOARCH
		bctx.InstallSuffix = "js"
		if installSuffix != "" {
			bctx.InstallSuffix += "_" + installSuffix
		}
	case "math/big":
		// Use pure Go version of math/big; we don't want non-Go assembly versions.
		bctx.BuildTags = append(bctx.BuildTags, "math_big_pure_go")
	case "crypto/x509", "os/user":
		// These stdlib packages have cgo and non-cgo versions (via build tags); we want the latter.
		bctx.CgoEnabled = false
	case "github.com/gopherjs/gopherjs/js", "github.com/gopherjs/gopherjs/nosync":
		// These packages are already embedded via gopherjspkg.FS virtual filesystem (which can be
		// safely vendored). Don't try to use vendor directory to resolve them.

		// TODO work ouw whether this is still critical in GOPATH mode
		// mode |= build.IgnoreVendor

		isVirtual = true
	}

	var pkg *build.Package
	var err error
	if s.modLookup == nil {
		pkg, err = bctx.Import(path, srcDir, mode)
		if err != nil {
			return nil, err
		}
	} else {
		dir, ok := s.modLookup[path]
		if !ok {
			return nil, fmt.Errorf("failed to find import directory for %v", path)
		}

		// set IgnoreVendor even in module mode to prevent go/build from doing
		// anything with go list; we've already done that work.
		pkg, err = bctx.ImportDir(dir, mode|build.IgnoreVendor)
		if err != nil {
			return nil, fmt.Errorf("build context ImportDir failed: %v", err)
		}
		// because ImportDir doesn't know the ImportPath, we need to set
		// certain things manually
		gp := filepath.SplitList(build.Default.GOPATH)[0]
		pkg.ImportPath = path
		pkg.BinDir = filepath.Join(gp, "bin")
		if !pkg.IsCommand() {
			pkg.PkgObj = filepath.Join(gp, "pkg", build.Default.GOOS+"_js", path+".a")
		}
	}

	switch path {
	case "os":
		pkg.GoFiles = excludeExecutable(pkg.GoFiles) // Need to exclude executable implementation files, because some of them contain package scope variables that perform (indirectly) syscalls on init.
	case "runtime":
		pkg.GoFiles = []string{"error.go"}
	case "runtime/internal/sys":
		pkg.GoFiles = []string{fmt.Sprintf("zgoos_%s.go", bctx.GOOS), "zversion.go"}
	case "runtime/pprof":
		pkg.GoFiles = nil
	case "internal/poll":
		pkg.GoFiles = exclude(pkg.GoFiles, "fd_poll_runtime.go")
	case "crypto/rand":
		pkg.GoFiles = []string{"rand.go", "util.go"}
		pkg.TestGoFiles = exclude(pkg.TestGoFiles, "rand_linux_test.go") // Don't want linux-specific tests (since linux-specific package files are excluded too).
	}

	if len(pkg.CgoFiles) > 0 {
		return nil, &ImportCError{path}
	}

	if pkg.IsCommand() {
		pkg.PkgObj = filepath.Join(pkg.BinDir, filepath.Base(pkg.ImportPath)+".js")
	}

	// this is pre-module behaviour. Don't touch it
	if s.modLookup == nil {
		if _, err := os.Stat(pkg.PkgObj); os.IsNotExist(err) && strings.HasPrefix(pkg.PkgObj, build.Default.GOROOT) {
			// fall back to GOPATH
			firstGopathWorkspace := filepath.SplitList(build.Default.GOPATH)[0] // TODO: Need to check inside all GOPATH workspaces.
			gopathPkgObj := filepath.Join(firstGopathWorkspace, pkg.PkgObj[len(build.Default.GOROOT):])
			if _, err := os.Stat(gopathPkgObj); err == nil {
				pkg.PkgObj = gopathPkgObj
			}
		}
	}

	jsFiles, err := jsFilesFromDir(&bctx, pkg.Dir)
	if err != nil {
		return nil, err
	}

	return &PackageData{Package: pkg, JSFiles: jsFiles, IsVirtual: isVirtual}, nil
}

// excludeExecutable excludes all executable implementation .go files.
// They have "executable_" prefix.
func excludeExecutable(goFiles []string) []string {
	var s []string
	for _, f := range goFiles {
		if strings.HasPrefix(f, "executable_") {
			continue
		}
		s = append(s, f)
	}
	return s
}

// exclude returns files, excluding specified files.
func exclude(files []string, exclude ...string) []string {
	var s []string
Outer:
	for _, f := range files {
		for _, e := range exclude {
			if f == e {
				continue Outer
			}
		}
		s = append(s, f)
	}
	return s
}

// ImportDir is like Import but processes the Go package found in the named
// directory.
func ImportDir(dir string, mode build.ImportMode, installSuffix string, buildTags []string) (*PackageData, error) {
	bctx := NewBuildContext(installSuffix, buildTags)
	pkg, err := bctx.ImportDir(dir, mode)
	if err != nil {
		return nil, err
	}

	jsFiles, err := jsFilesFromDir(bctx, pkg.Dir)
	if err != nil {
		return nil, err
	}

	return &PackageData{Package: pkg, JSFiles: jsFiles}, nil
}

// parseAndAugment parses and returns all .go files of given pkg.
// Standard Go library packages are augmented with files in compiler/natives folder.
// If isTest is true and pkg.ImportPath has no _test suffix, package is built for running internal tests.
// If isTest is true and pkg.ImportPath has _test suffix, package is built for running external tests.
//
// The native packages are augmented by the contents of natives.FS in the following way.
// The file names do not matter except the usual `_test` suffix. The files for
// native overrides get added to the package (even if they have the same name
// as an existing file from the standard library). For all identifiers that exist
// in the original AND the overrides, the original identifier in the AST gets
// replaced by `_`. New identifiers that don't exist in original package get added.
func parseAndAugment(bctx *build.Context, pkg *build.Package, isTest bool, fileSet *token.FileSet, hw io.Writer) ([]*ast.File, error) {
	var files []*ast.File
	replacedDeclNames := make(map[string]bool)
	funcName := func(d *ast.FuncDecl) string {
		if d.Recv == nil || len(d.Recv.List) == 0 {
			return d.Name.Name
		}
		recv := d.Recv.List[0].Type
		if star, ok := recv.(*ast.StarExpr); ok {
			recv = star.X
		}
		return recv.(*ast.Ident).Name + "." + d.Name.Name
	}
	isXTest := strings.HasSuffix(pkg.ImportPath, "_test")
	importPath := pkg.ImportPath
	if isXTest {
		importPath = importPath[:len(importPath)-5]
	}

	nativesContext := &build.Context{
		GOROOT:   "/",
		GOOS:     build.Default.GOOS,
		GOARCH:   "js",
		Compiler: "gc",
		JoinPath: path.Join,
		SplitPathList: func(list string) []string {
			if list == "" {
				return nil
			}
			return strings.Split(list, "/")
		},
		IsAbsPath: path.IsAbs,
		IsDir: func(name string) bool {
			dir, err := natives.FS.Open(name)
			if err != nil {
				return false
			}
			defer dir.Close()
			info, err := dir.Stat()
			if err != nil {
				return false
			}
			return info.IsDir()
		},
		HasSubdir: func(root, name string) (rel string, ok bool) {
			panic("not implemented")
		},
		ReadDir: func(name string) (fi []os.FileInfo, err error) {
			dir, err := natives.FS.Open(name)
			if err != nil {
				return nil, err
			}
			defer dir.Close()
			return dir.Readdir(0)
		},
		OpenFile: func(name string) (r io.ReadCloser, err error) {
			return natives.FS.Open(name)
		},
	}

	// reflect needs to tell Go 1.11 apart from Go 1.11.1 for https://github.com/gopherjs/gopherjs/issues/862,
	// so provide it with the custom go1.11.1 build tag whenever we're on Go 1.11.1 or later.
	// TODO: Remove this ad hoc special behavior in GopherJS 1.12.
	if runtime.Version() != "go1.11" {
		nativesContext.ReleaseTags = append(nativesContext.ReleaseTags, "go1.11.1")
	}

	if nativesPkg, err := nativesContext.Import(importPath, "", 0); err == nil {
		names := nativesPkg.GoFiles
		if isTest {
			names = append(names, nativesPkg.TestGoFiles...)
		}
		if isXTest {
			names = nativesPkg.XTestGoFiles
		}
		for _, name := range names {
			fullPath := path.Join(nativesPkg.Dir, name)
			r, err := nativesContext.OpenFile(fullPath)
			if err != nil {
				panic(err)
			}
			rbyts, err := ioutil.ReadAll(r)
			r.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read native file %v: %v", fullPath, err)
			}
			if hw != nil {
				fmt.Fprintf(hw, "file: %v\n", fullPath)
				fmt.Fprintf(hw, "%s\n", rbyts)
				fmt.Fprintf(hw, "%d bytes\n", len(rbyts))
			}
			file, err := parser.ParseFile(fileSet, fullPath, rbyts, parser.ParseComments)
			if err != nil {
				panic(err)
			}
			for _, decl := range file.Decls {
				switch d := decl.(type) {
				case *ast.FuncDecl:
					replacedDeclNames[funcName(d)] = true
				case *ast.GenDecl:
					switch d.Tok {
					case token.TYPE:
						for _, spec := range d.Specs {
							replacedDeclNames[spec.(*ast.TypeSpec).Name.Name] = true
						}
					case token.VAR, token.CONST:
						for _, spec := range d.Specs {
							for _, name := range spec.(*ast.ValueSpec).Names {
								replacedDeclNames[name.Name] = true
							}
						}
					}
				}
			}
			files = append(files, file)
		}
	}
	delete(replacedDeclNames, "init")

	var errList compiler.ErrorList
	for _, name := range pkg.GoFiles {
		if !filepath.IsAbs(name) { // name might be absolute if specified directly. E.g., `gopherjs build /abs/file.go`.
			name = filepath.Join(pkg.Dir, name)
		}
		r, err := buildutil.OpenFile(bctx, name)
		if err != nil {
			return nil, err
		}
		rbyts, err := ioutil.ReadAll(r)
		r.Close()
		if err != nil {
			return nil, err
		}
		if hw != nil {
			fmt.Fprintf(hw, "file: %v\n", name)
			fmt.Fprintf(hw, "%s\n", rbyts)
			fmt.Fprintf(hw, "%d bytes\n", len(rbyts))
		}
		file, err := parser.ParseFile(fileSet, name, rbyts, parser.ParseComments)
		if err != nil {
			if list, isList := err.(scanner.ErrorList); isList {
				if len(list) > 10 {
					list = append(list[:10], &scanner.Error{Pos: list[9].Pos, Msg: "too many errors"})
				}
				for _, entry := range list {
					errList = append(errList, entry)
				}
				continue
			}
			errList = append(errList, err)
			continue
		}

		switch pkg.ImportPath {
		case "crypto/rand", "encoding/gob", "encoding/json", "expvar", "go/token", "log", "math/big", "math/rand", "regexp", "testing", "time":
			for _, spec := range file.Imports {
				path, _ := strconv.Unquote(spec.Path.Value)
				if path == "sync" {
					if spec.Name == nil {
						spec.Name = ast.NewIdent("sync")
					}
					spec.Path.Value = `"github.com/gopherjs/gopherjs/nosync"`
				}
			}
		}

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if replacedDeclNames[funcName(d)] {
					d.Name = ast.NewIdent("_")
				}
			case *ast.GenDecl:
				switch d.Tok {
				case token.TYPE:
					for _, spec := range d.Specs {
						s := spec.(*ast.TypeSpec)
						if replacedDeclNames[s.Name.Name] {
							s.Name = ast.NewIdent("_")
						}
					}
				case token.VAR, token.CONST:
					for _, spec := range d.Specs {
						s := spec.(*ast.ValueSpec)
						for i, name := range s.Names {
							if replacedDeclNames[name.Name] {
								s.Names[i] = ast.NewIdent("_")
							}
						}
					}
				}
			}
		}
		files = append(files, file)
	}
	if errList != nil {
		return nil, errList
	}
	return files, nil
}

type Options struct {
	GOROOT         string
	GOPATH         string
	Verbose        bool
	Quiet          bool
	Watch          bool
	CreateMapFile  bool
	MapToLocalDisk bool
	Minify         bool
	Color          bool
	BuildTags      []string
}

func (o *Options) PrintError(format string, a ...interface{}) {
	if o.Color {
		format = "\x1B[31m" + format + "\x1B[39m"
	}
	fmt.Fprintf(os.Stderr, format, a...)
}

func (o *Options) PrintSuccess(format string, a ...interface{}) {
	if o.Color {
		format = "\x1B[32m" + format + "\x1B[39m"
	}
	fmt.Fprintf(os.Stderr, format, a...)
}

type PackageData struct {
	*build.Package
	JSFiles   []string
	IsTest    bool // IsTest is true if the package is being built for running tests.
	UpToDate  bool
	IsVirtual bool // If true, the package does not have a corresponding physical directory on disk.
}

type Session struct {
	options      *Options
	bctx         *build.Context
	Archives     map[string]*compiler.Archive
	Types        map[string]*types.Package
	Watcher      *fsnotify.Watcher
	wd           string // working directory
	buildCache   *cache.Cache
	didCacheWork bool

	// map of import path to dir for module mode resolution
	// a nil value implies we are not in module mode
	modLookup map[string]string

	// map of module path
	mods map[string]string
}

func NewSession(options *Options, tests bool, imports ...string) (*Session, error) {
	if options.GOROOT == "" {
		options.GOROOT = build.Default.GOROOT
	}
	if options.GOPATH == "" {
		options.GOPATH = build.Default.GOPATH
	}
	options.Verbose = options.Verbose || options.Watch

	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %v", err)
	}

	ucd, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine user cache dir: %v", err)
	}

	buildCacheDir := filepath.Join(ucd, buildCacheDirName)
	if err := os.MkdirAll(buildCacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create build cache dir %v: %v", buildCacheDir, err)
	}

	buildCache, err := cache.Open(buildCacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open build cache dir: %v", err)
	}

	s := &Session{
		options:    options,
		Archives:   make(map[string]*compiler.Archive),
		wd:         wd,
		buildCache: buildCache,
	}
	s.bctx = NewBuildContext(s.InstallSuffix(), s.options.BuildTags)
	s.Types = make(map[string]*types.Package)
	if err := s.determineModLookup(tests, imports); err != nil {
		return nil, err
	}
	if options.Watch {
		if out, err := exec.Command("ulimit", "-n").Output(); err == nil {
			if n, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil && n < 1024 {
				fmt.Printf("Warning: The maximum number of open file descriptors is very low (%d). Change it with 'ulimit -n 8192'.\n", n)
			}
		}

		var err error
		s.Watcher, err = fsnotify.NewWatcher()
		if err != nil {
			return nil, fmt.Errorf("failed to create watcher: %v", err)
		}
	}
	return s, nil
}

func (s *Session) GO111MODULE() bool {
	return s.modLookup != nil
}

func (s *Session) determineModLookup(tests bool, imports []string) error {
	goenvCmd := exec.Command("go", "env", "GOMOD")
	output, err := goenvCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to determine if we are module-mode or not: %v", err)
	}
	if strings.TrimSpace(string(output)) == "" {
		return nil
	}

	if tests {
		if len(imports) != 1 {
			panic("invariant broken for test list")
		}
		imports = append(imports, "testing", "testing/internal/testdeps")
	}

	// we always need to be able to resolve these
	imports = append(imports, "runtime", "github.com/gopherjs/gopherjs/js", "github.com/gopherjs/gopherjs/nosync")

	var stdout, stderr bytes.Buffer
	golistCmd := exec.Command("go", "list", "-deps", `-f={{if or (eq .ForTest "") (eq .ForTest "`+imports[0]+`")}}{"ImportPath": "{{.ImportPath}}", "Dir": "{{.Dir}}"{{with .Module}}, "Module": {"Path": "{{.Path}}", "Dir": "{{.Dir}}"}{{end}}}{{end}}`)
	if tests {
		golistCmd.Args = append(golistCmd.Args, "-test")
	}
	if len(s.bctx.BuildTags) > 0 {
		golistCmd.Args = append(golistCmd.Args, "-tags="+strings.Join(s.bctx.BuildTags, " "))
	}
	golistCmd.Args = append(golistCmd.Args, imports...)
	golistCmd.Stdout = &stdout
	golistCmd.Stderr = &stderr

	if err := golistCmd.Run(); err != nil {
		return fmt.Errorf("failed to run %v: %v\n%s", strings.Join(golistCmd.Args, " "), err, stderr.Bytes())
	}

	dec := json.NewDecoder(&stdout)

	s.modLookup = make(map[string]string)
	s.mods = make(map[string]string)

	for {
		var entry struct {
			ImportPath string
			Dir        string
			Module     struct {
				Path string
				Dir  string
			}
		}

		if err := dec.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode list output: %v\n%s", err, stdout.Bytes())
		}

		ipParts := strings.Split(entry.ImportPath, " ")
		entry.ImportPath = ipParts[0]

		s.modLookup[entry.ImportPath] = entry.Dir
		s.mods[entry.Module.Path] = entry.Module.Dir
	}

	return nil
}

func (s *Session) IsModulePath(path string) (string, bool) {
	dir, ok := s.mods[path]
	return dir, ok
}

func (s *Session) Cleanup() error {
	if s.didCacheWork {
		s.buildCache.Trim()
	}
	return nil
}

// BuildContext returns the session's build context.
func (s *Session) BuildContext() *build.Context { return s.bctx }

func (s *Session) InstallSuffix() string {
	if s.options.Minify {
		return "min"
	}
	return ""
}

func (s *Session) BuildDir(packagePath string, importPath string, pkgObj string) error {
	if s.Watcher != nil {
		s.Watcher.Add(packagePath)
	}
	buildPkg, err := s.bctx.ImportDir(packagePath, 0)
	if err != nil {
		return err
	}
	pkg := &PackageData{Package: buildPkg}
	jsFiles, err := jsFilesFromDir(s.bctx, pkg.Dir)
	if err != nil {
		return err
	}
	pkg.JSFiles = jsFiles
	archive, err := s.BuildPackage(pkg)
	if err != nil {
		return err
	}
	if pkgObj == "" {
		pkgObj = filepath.Base(packagePath) + ".js"
	}
	if pkg.IsCommand() && !pkg.UpToDate {
		if err := s.WriteCommandPackage(archive, pkgObj); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) BuildFiles(filenames []string, pkgObj string, packagePath string) error {
	pkg := &PackageData{
		Package: &build.Package{
			Name:       "main",
			ImportPath: "main",
			Dir:        packagePath,
		},
	}

	for _, file := range filenames {
		if strings.HasSuffix(file, ".inc.js") {
			pkg.JSFiles = append(pkg.JSFiles, file)
			continue
		}
		pkg.GoFiles = append(pkg.GoFiles, file)
	}

	archive, err := s.BuildPackage(pkg)
	if err != nil {
		return err
	}
	if s.Types["main"].Name() != "main" {
		return fmt.Errorf("cannot build/run non-main package")
	}
	return s.WriteCommandPackage(archive, pkgObj)
}

func (s *Session) BuildImportPath(path string) (*PackageData, *compiler.Archive, error) {
	return s.buildImportPathWithSrcDir(path, s.wd)
}

func (s *Session) buildImportPathWithSrcDir(path string, srcDir string) (*PackageData, *compiler.Archive, error) {
	pkg, err := s.importWithSrcDir(*s.bctx, path, srcDir, 0, s.InstallSuffix())
	if s.Watcher != nil && pkg != nil { // add watch even on error
		s.Watcher.Add(pkg.Dir)
	}
	if err != nil {
		return nil, nil, err
	}

	archive, err := s.BuildPackage(pkg)
	if err != nil {
		return nil, nil, err
	}

	return pkg, archive, nil
}

func hashCompilerBinary() (string, error) {
	if compilerBinaryHash != "" {
		return compilerBinaryHash, nil
	}

	binHash := sha256.New()
	binPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not locate GopherJS binary: %v", err)
	}
	binFile, err := os.Open(binPath)
	if err != nil {
		return "", fmt.Errorf("could not open %v: %v", binPath, err)
	}
	defer binFile.Close()
	if _, err := io.Copy(binHash, binFile); err != nil {
		return "", fmt.Errorf("failed to hash %v: %v", binPath, err)
	}
	compilerBinaryHash = fmt.Sprintf("%#x", binHash.Sum(nil))
	return compilerBinaryHash, nil
}

func (s *Session) BuildPackage(pkg *PackageData) (*compiler.Archive, error) {
	// Instead of calculation the reverse dependency graph up front, we instead
	// walk the import graph recursively (see call to
	// s.buildImportPathWithSrcDir below). Session.Archives keeps track of those
	// packages we have already seen.
	if archive, ok := s.Archives[pkg.ImportPath]; ok {
		return archive, nil
	}

	var pkgHash *cache.Hash
	var hw io.Writer
	var hashDebugOut *bytes.Buffer

	// We never cache main or test packages because of the "hack" used to run
	// arbitrary files and some legacy (at the time of writing) unknown reason
	// for test packages.
	//
	// Where we do cache an archive, build up a hash that represents a complete
	// description of a repeatable computation (command line, environment
	// variables, input file contents, executable contents). This therefore
	// needs to be a stable computation.  The iteration through imports is, by
	// definition, stable, because those imports are ordered.
	if !(pkg.IsCommand() || pkg.IsTest) {
		pkgHash = cache.NewHash("## build " + pkg.ImportPath)
		hw = pkgHash
		if hashDebug {
			hashDebugOut = new(bytes.Buffer)
			hw = io.MultiWriter(hashDebugOut, pkgHash)
		}
	}

	if hw != nil {
		fmt.Fprintf(hw, "compiler binary hash: %v\n", compilerBinaryHash)

		orderedBuildTags := append([]string{}, s.options.BuildTags...)
		sort.Strings(orderedBuildTags)

		fmt.Fprintf(hw, "build tags: %v\n", strings.Join(orderedBuildTags, ","))

		for _, importedPkgPath := range pkg.Imports {
			// Ignore all imports that aren't mentioned in import specs of pkg. For
			// example, this ignores imports such as runtime/internal/sys and
			// runtime/internal/atomic; nobody explicitly adds such imports to their
			// packages, they are automatically added by the Go tool.
			//
			// TODO perhaps there is a cleaner way of doing this?
			ignored := true
			for _, pos := range pkg.ImportPos[importedPkgPath] {
				importFile := filepath.Base(pos.Filename)
				for _, file := range pkg.GoFiles {
					if importFile == file {
						ignored = false
						break
					}
				}
				if !ignored {
					break
				}
			}

			if importedPkgPath == "unsafe" || ignored {
				continue
			}

			_, importedArchive, err := s.buildImportPathWithSrcDir(importedPkgPath, s.wd)
			if err != nil {
				return nil, err
			}

			fmt.Fprintf(hw, "import: %v\n", importedPkgPath)
			fmt.Fprintf(hw, "  hash: %#x\n", importedArchive.Hash)
		}
	}

	fset := token.NewFileSet()
	files, err := parseAndAugment(s.bctx, pkg.Package, pkg.IsTest, fset, hw)
	if err != nil {
		return nil, err
	}

	if hw != nil {
		for _, name := range pkg.JSFiles {
			hashFile := func() error {
				fp := filepath.Join(pkg.Dir, name)
				file, err := s.bctx.OpenFile(fp)
				if err != nil {
					return fmt.Errorf("failed to open %v: %v", fp, err)
				}
				defer file.Close()
				fmt.Fprintf(hw, "file: %v\n", fp)
				n, err := io.Copy(hw, file)
				if err != nil {
					return fmt.Errorf("failed to hash file contents: %v", err)
				}
				fmt.Fprintf(hw, "%d bytes\n", n)
				return nil
			}

			if err := hashFile(); err != nil {
				return nil, fmt.Errorf("failed to hash file %v: %v", name, err)
			}
		}

		if hashDebug {
			fmt.Printf("%s", hashDebugOut.String())
		}

		// At this point we have a complete Hash. Hence we can check the Cache to see whether
		// we already have an archive for this key.

		if objFilePath, _, err := s.buildCache.GetFile(pkgHash.Sum()); err == nil {
			// Try to open objFile; we are not guaranteed it will still be available
			objFile, err := os.Open(objFilePath)
			if err == nil {
				archive, err := compiler.ReadArchive(pkg.PkgObj, pkg.ImportPath, objFile, s.Types)
				objFile.Close()
				if err == nil {
					s.Archives[pkg.ImportPath] = archive
					return archive, nil
				}
			}
		}
	}

	// At this point, for whatever reason, we were unable to read a build-cached archive.
	// So we need to build one.

	if s.options.Verbose {
		fmt.Fprintf(os.Stderr, "Cache miss for %v\n", pkg.ImportPath)
	}

	localImportPathCache := make(map[string]*compiler.Archive)
	importContext := &compiler.ImportContext{
		Packages: s.Types,
		Import: func(path string) (*compiler.Archive, error) {
			if archive, ok := localImportPathCache[path]; ok {
				return archive, nil
			}
			_, archive, err := s.buildImportPathWithSrcDir(path, s.wd)
			if err != nil {
				return nil, err
			}
			localImportPathCache[path] = archive
			return archive, nil
		},
	}
	archive, err := compiler.Compile(pkg.ImportPath, files, fset, importContext, s.options.Minify)
	if err != nil {
		return nil, err
	}

	for _, jsFile := range pkg.JSFiles {
		code, err := ioutil.ReadFile(filepath.Join(pkg.Dir, jsFile))
		if err != nil {
			return nil, err
		}
		archive.IncJSCode = append(archive.IncJSCode, []byte("\t(function() {\n")...)
		archive.IncJSCode = append(archive.IncJSCode, code...)
		archive.IncJSCode = append(archive.IncJSCode, []byte("\n\t}).call($global);\n")...)
	}

	// Mark this import path as having been "seen"
	s.Archives[pkg.ImportPath] = archive

	if pkgHash != nil {
		archive.Hash = pkgHash.Sum()

		var buf bytes.Buffer
		if err := compiler.WriteArchive(archive, &buf); err != nil {
			return nil, fmt.Errorf("failed to write archive: %v", err)
		}

		if _, _, err := s.buildCache.Put(pkgHash.Sum(), bytes.NewReader(buf.Bytes())); err != nil {
			return nil, fmt.Errorf("failed to cache archive: %v", err)
		}

		s.didCacheWork = true
	}

	return archive, nil
}

func (s *Session) WriteCommandPackage(archive *compiler.Archive, pkgObj string) error {
	if err := os.MkdirAll(filepath.Dir(pkgObj), 0777); err != nil {
		return err
	}
	codeFile, err := os.Create(pkgObj)
	if err != nil {
		return err
	}
	defer codeFile.Close()

	sourceMapFilter := &compiler.SourceMapFilter{Writer: codeFile}
	if s.options.CreateMapFile {
		m := &sourcemap.Map{File: filepath.Base(pkgObj)}
		mapFile, err := os.Create(pkgObj + ".map")
		if err != nil {
			return err
		}

		defer func() {
			m.WriteTo(mapFile)
			mapFile.Close()
			fmt.Fprintf(codeFile, "//# sourceMappingURL=%s.map\n", filepath.Base(pkgObj))
		}()

		sourceMapFilter.MappingCallback = NewMappingCallback(m, s.options.GOROOT, s.options.GOPATH, s.options.MapToLocalDisk)
	}

	deps, err := compiler.ImportDependencies(archive, func(path string) (*compiler.Archive, error) {
		if archive, ok := s.Archives[path]; ok {
			return archive, nil
		}
		_, archive, err := s.buildImportPathWithSrcDir(path, s.wd)
		return archive, err
	})
	if err != nil {
		return err
	}
	return compiler.WriteProgramCode(deps, sourceMapFilter)
}

func NewMappingCallback(m *sourcemap.Map, goroot, gopath string, localMap bool) func(generatedLine, generatedColumn int, originalPos token.Position) {
	return func(generatedLine, generatedColumn int, originalPos token.Position) {
		if !originalPos.IsValid() {
			m.AddMapping(&sourcemap.Mapping{GeneratedLine: generatedLine, GeneratedColumn: generatedColumn})
			return
		}

		file := originalPos.Filename

		switch hasGopathPrefix, prefixLen := hasGopathPrefix(file, gopath); {
		case localMap:
			// no-op:  keep file as-is
		case hasGopathPrefix:
			file = filepath.ToSlash(file[prefixLen+4:])
		case strings.HasPrefix(file, goroot):
			file = filepath.ToSlash(file[len(goroot)+4:])
		default:
			file = filepath.Base(file)
		}

		m.AddMapping(&sourcemap.Mapping{GeneratedLine: generatedLine, GeneratedColumn: generatedColumn, OriginalFile: file, OriginalLine: originalPos.Line, OriginalColumn: originalPos.Column})
	}
}

func jsFilesFromDir(bctx *build.Context, dir string) ([]string, error) {
	files, err := buildutil.ReadDir(bctx, dir)
	if err != nil {
		return nil, err
	}
	var jsFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".inc.js") && file.Name()[0] != '_' && file.Name()[0] != '.' {
			jsFiles = append(jsFiles, file.Name())
		}
	}
	return jsFiles, nil
}

// hasGopathPrefix returns true and the length of the matched GOPATH workspace,
// iff file has a prefix that matches one of the GOPATH workspaces.
func hasGopathPrefix(file, gopath string) (hasGopathPrefix bool, prefixLen int) {
	gopathWorkspaces := filepath.SplitList(gopath)
	for _, gopathWorkspace := range gopathWorkspaces {
		gopathWorkspace = filepath.Clean(gopathWorkspace)
		if strings.HasPrefix(file, gopathWorkspace) {
			return true, len(gopathWorkspace)
		}
	}
	return false, 0
}

func (s *Session) WaitForChange() {
	s.options.PrintSuccess("watching for changes...\n")
	for {
		select {
		case ev := <-s.Watcher.Events:
			if ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) == 0 || filepath.Base(ev.Name)[0] == '.' {
				continue
			}
			if !strings.HasSuffix(ev.Name, ".go") && !strings.HasSuffix(ev.Name, ".inc.js") {
				continue
			}
			s.options.PrintSuccess("change detected: %s\n", ev.Name)
		case err := <-s.Watcher.Errors:
			s.options.PrintError("watcher error: %s\n", err.Error())
		}
		break
	}

	go func() {
		for range s.Watcher.Events {
			// consume, else Close() may deadlock
		}
	}()
	s.Watcher.Close()
}

func ImportPaths(vs ...string) ([]string, error) {
	if len(vs) == 0 {
		vs = []string{"."}
	}

	args := []string{"go", "list"}
	args = append(args, vs...)
	cmd := exec.Command(args[0], args[1:]...)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to resolve import paths: %v\n%s", err, stderr.String())
	}

	res := strings.Split(strings.TrimSpace(stdout.String()), "\n")

	for i, v := range res {
		// inefficiently handles CR
		res[i] = strings.TrimSpace(v)
	}

	return res, nil
}
