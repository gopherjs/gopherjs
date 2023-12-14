// Package build implements GopherJS build system.
//
// WARNING: This package's API is treated as internal and currently doesn't
// provide any API stability guarantee, use it at your own risk. If you need a
// stable interface, prefer invoking the gopherjs CLI tool as a subprocess.
package build

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gopherjs/gopherjs/compiler"
	"github.com/gopherjs/gopherjs/compiler/astutil"
	log "github.com/sirupsen/logrus"

	"github.com/neelance/sourcemap"
	"golang.org/x/tools/go/buildutil"

	"github.com/gopherjs/gopherjs/build/cache"
)

// DefaultGOROOT is the default GOROOT value for builds.
//
// It uses the GOPHERJS_GOROOT environment variable if it is set,
// or else the default GOROOT value of the system Go distribution.
var DefaultGOROOT = func() string {
	if goroot, ok := os.LookupEnv("GOPHERJS_GOROOT"); ok {
		// GopherJS-specific GOROOT value takes precedence.
		return goroot
	}
	// The usual default GOROOT.
	return build.Default.GOROOT
}()

// NewBuildContext creates a build context for building Go packages
// with GopherJS compiler.
//
// Core GopherJS packages (i.e., "github.com/gopherjs/gopherjs/js", "github.com/gopherjs/gopherjs/nosync")
// are loaded from gopherjspkg.FS virtual filesystem if not present in GOPATH or
// go.mod.
func NewBuildContext(installSuffix string, buildTags []string) XContext {
	e := DefaultEnv()
	e.InstallSuffix = installSuffix
	e.BuildTags = buildTags
	realGOROOT := goCtx(e)
	return &chainedCtx{
		primary:   realGOROOT,
		secondary: gopherjsCtx(e),
	}
}

// Import returns details about the Go package named by the import path. If the
// path is a local import path naming a package that can be imported using
// a standard import path, the returned package will set p.ImportPath to
// that path.
//
// In the directory containing the package, .go and .inc.js files are
// considered part of the package except for:
//
//   - .go files in package documentation
//   - files starting with _ or . (likely editor temporary files)
//   - files with build constraints not satisfied by the context
//
// If an error occurs, Import returns a non-nil error and a nil
// *PackageData.
func Import(path string, mode build.ImportMode, installSuffix string, buildTags []string) (*PackageData, error) {
	wd, err := os.Getwd()
	if err != nil {
		// Getwd may fail if we're in GOOS=js mode. That's okay, handle
		// it by falling back to empty working directory. It just means
		// Import will not be able to resolve relative import paths.
		wd = ""
	}
	xctx := NewBuildContext(installSuffix, buildTags)
	return xctx.Import(path, wd, mode)
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
	xctx := NewBuildContext(installSuffix, buildTags)
	pkg, err := xctx.Import(".", dir, mode)
	if err != nil {
		return nil, err
	}

	return pkg, nil
}

// overrideInfo is used by parseAndAugment methods to manage
// directives and how the overlay and original are merged.
type overrideInfo struct {
	// KeepOriginal indicates that the original code should be kept
	// but the identifier will be prefixed by `_gopherjs_original_foo`.
	keepOriginal bool

	// purgeMethods indicates that this info is for a type and
	// if a method has this type as a receiver should also be removed.
	// If the method is defined in the overlays and therefore has its
	// own overrides, this will be ignored.
	purgeMethods bool
}

// parseAndAugment parses and returns all .go files of given pkg.
// Standard Go library packages are augmented with files in compiler/natives folder.
// If isTest is true and pkg.ImportPath has no _test suffix, package is built for running internal tests.
// If isTest is true and pkg.ImportPath has _test suffix, package is built for running external tests.
//
// The native packages are augmented by the contents of natives.FS in the following way.
// The file names do not matter except the usual `_test` suffix. The files for
// native overrides get added to the package (even if they have the same name
// as an existing file from the standard library).
//
//   - For function identifiers that exist in the original and the overrides and have the
//     directive `gopherjs:keep-original`, the original identifier in the AST gets
//     prefixed by `_gopherjs_original_`.
//   - For identifiers that exist in the original and the overrides
//     and have the directive `gopherjs:purge`, both the original and override are removed.
//     This is for completely removing something which is currently invalid for GopherJS.
//     For any purged types any methods with that type as the receiver are also removed.
//   - Otherwise for identifiers that exist in the original and the overrides,
//     the original identifier is removed.
//   - New identifiers that don't exist in original package get added.
func parseAndAugment(xctx XContext, pkg *PackageData, isTest bool, fileSet *token.FileSet) ([]*ast.File, []JSFile, error) {
	jsFiles, overlayFiles := parseOverlayFiles(xctx, pkg, isTest, fileSet)

	originalFiles, err := parserOriginalFiles(pkg, fileSet)
	if err != nil {
		return nil, nil, err
	}

	overrides := make(map[string]overrideInfo)
	for _, file := range overlayFiles {
		augmentOverlayFile(fileSet, file, overrides)
		pruneImports(fileSet, file)
	}
	delete(overrides, "init")

	for _, file := range originalFiles {
		augmentOriginalImports(pkg.ImportPath, file)
		augmentOriginalFile(fileSet, file, overrides)
		pruneImports(fileSet, file)
	}

	return append(overlayFiles, originalFiles...), jsFiles, nil
}

// parseOverlayFiles loads and parses overlay files
// to augment the original files with.
func parseOverlayFiles(xctx XContext, pkg *PackageData, isTest bool, fileSet *token.FileSet) ([]JSFile, []*ast.File) {
	isXTest := strings.HasSuffix(pkg.ImportPath, "_test")
	importPath := pkg.ImportPath
	if isXTest {
		importPath = importPath[:len(importPath)-5]
	}

	var jsFiles []JSFile
	var files []*ast.File
	nativesContext := overlayCtx(xctx.Env())
	if nativesPkg, err := nativesContext.Import(importPath, "", 0); err == nil {
		jsFiles = nativesPkg.JSFiles
		names := nativesPkg.GoFiles
		if isTest {
			names = append(names, nativesPkg.TestGoFiles...)
		}
		if isXTest {
			names = nativesPkg.XTestGoFiles
		}

		for _, name := range names {
			fullPath := path.Join(nativesPkg.Dir, name)
			r, err := nativesContext.bctx.OpenFile(fullPath)
			if err != nil {
				panic(err)
			}
			// Files should be uniquely named and in the original package directory in order to be
			// ordered correctly
			newPath := path.Join(pkg.Dir, "gopherjs__"+name)
			file, err := parser.ParseFile(fileSet, newPath, r, parser.ParseComments)
			if err != nil {
				panic(err)
			}
			r.Close()

			files = append(files, file)
		}
	}

	return jsFiles, files
}

// parserOriginalFiles loads and parses the original files to augment.
func parserOriginalFiles(pkg *PackageData, fileSet *token.FileSet) ([]*ast.File, error) {
	var files []*ast.File
	var errList compiler.ErrorList
	for _, name := range pkg.GoFiles {
		if !filepath.IsAbs(name) { // name might be absolute if specified directly. E.g., `gopherjs build /abs/file.go`.
			name = filepath.Join(pkg.Dir, name)
		}
		r, err := buildutil.OpenFile(pkg.bctx, name)
		if err != nil {
			return nil, err
		}
		file, err := parser.ParseFile(fileSet, name, r, parser.ParseComments)
		r.Close()
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

		files = append(files, file)
	}

	if errList != nil {
		return nil, errList
	}
	return files, nil
}

// augmentOverlayFile is the part of parseAndAugment that processes
// an overlay file AST to collect information such as compiler directives and
// perform any initial augmentation needed to the overlay.
func augmentOverlayFile(fileSet *token.FileSet, file *ast.File, overrides map[string]overrideInfo) {
	for i, decl := range file.Decls {
		purgeDecl := astutil.Purge(decl)
		switch d := decl.(type) {
		case *ast.FuncDecl:
			k := astutil.FuncKey(d)
			overrides[k] = overrideInfo{
				keepOriginal: astutil.KeepOriginal(d),
			}
		case *ast.GenDecl:
			for j, spec := range d.Specs {
				purgeSpec := purgeDecl || astutil.Purge(spec)
				switch s := spec.(type) {
				case *ast.TypeSpec:
					overrides[s.Name.Name] = overrideInfo{
						purgeMethods: purgeSpec,
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						overrides[name.Name] = overrideInfo{}
					}
				}
				if purgeSpec {
					d.Specs[j] = nil
				}
			}
		}
		if purgeDecl {
			file.Decls[i] = nil
		}
	}
	finalizeRemovals(file)
}

// augmentOriginalImports is the part of parseAndAugment that processes
// an original file AST to modify the imports for that file.
func augmentOriginalImports(importPath string, file *ast.File) {
	switch importPath {
	case "crypto/rand", "encoding/gob", "encoding/json", "expvar", "go/token", "log", "math/big", "math/rand", "regexp", "time":
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
}

// augmentOriginalFile is the part of parseAndAugment that processes an
// original file AST to augment the source code using the overrides from
// the overlay files.
func augmentOriginalFile(fileSet *token.FileSet, file *ast.File, overrides map[string]overrideInfo) {
	for i, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if info, ok := overrides[astutil.FuncKey(d)]; ok {
				if info.keepOriginal {
					// Allow overridden function calls
					// The standard library implementation of foo() becomes _gopherjs_original_foo()
					d.Name.Name = `_gopherjs_original_` + d.Name.Name
				} else {
					file.Decls[i] = nil
				}
			} else if recvKey := astutil.FuncReceiverKey(d); len(recvKey) > 0 {
				// check if the receiver has been purged, if so, remove the method too.
				if info, ok := overrides[recvKey]; ok && info.purgeMethods {
					file.Decls[i] = nil
				}
			}
		case *ast.GenDecl:
			for j, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if _, ok := overrides[s.Name.Name]; ok {
						d.Specs[j] = nil
					}
				case *ast.ValueSpec:
					if len(s.Names) == len(s.Values) {
						// multi-value context
						// e.g. var a, b = 2, 3
						for k, name := range s.Names {
							if _, ok := overrides[name.Name]; ok {
								s.Names[k] = nil
								s.Values[k] = nil
							}
						}
					} else {
						// single-value context
						// e.g. var a, b = func() (int, int) { return 2, 3 }()
						nameRemoved := false
						for _, name := range s.Names {
							if _, ok := overrides[name.Name]; ok {
								nameRemoved = true
								name.Name = `_`
							}
						}
						if nameRemoved {
							removeSpec := true
							for _, name := range s.Names {
								if name.Name != `_` {
									removeSpec = false
									break
								}
							}
							if removeSpec {
								d.Specs[j] = nil
							}
						}
					}
				}
			}
		}
	}
	finalizeRemovals(file)
}

// pruneImports will remove any unused imports from the file.
//
// This will not remove any dot (`.`) or blank (`_`) imports.
func pruneImports(fileSet *token.FileSet, file *ast.File) {
	unused := make(map[string]int, len(file.Imports))
	for i, in := range file.Imports {
		if name := astutil.ImportName(in); len(name) > 0 {
			unused[name] = i
		}
	}

	// Remove any import which is used in a selector.
	ast.Walk(astutil.NewCallbackVisitor(func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Obj == nil {
				delete(unused, id.Name)
			}
		}
		return len(unused) > 0
	}), file)

	if len(unused) == 0 {
		return
	}

	// Remove all import specifications
	isUnusedSpec := map[*ast.ImportSpec]bool{}
	for _, index := range unused {
		isUnusedSpec[file.Imports[index]] = true
	}
	for _, decl := range file.Decls {
		if d, ok := decl.(*ast.GenDecl); ok {
			for i, spec := range d.Specs {
				if other, ok := spec.(*ast.ImportSpec); ok && isUnusedSpec[other] {
					d.Specs[i] = nil
				}
			}
		}
	}

	// Remove the import copies in the file
	for _, index := range unused {
		file.Imports[index] = nil
	}

	finalizeRemovals(file)
}

// finalizeRemovals fully removes any declaration, specification, imports
// that have been set to nil. This will also remove the file's top-level
// comment group to remove any unassociated comments, including the comments
// from removed code.
func finalizeRemovals(file *ast.File) {
	fileChanged := false
	for i, decl := range file.Decls {
		switch d := decl.(type) {
		case nil:
			fileChanged = true
		case *ast.GenDecl:
			declChanged := false
			for j, spec := range d.Specs {
				switch s := spec.(type) {
				case nil:
					declChanged = true
				case *ast.ValueSpec:
					specChanged := false
					for _, name := range s.Names {
						if name == nil {
							specChanged = true
							break
						}
					}
					if specChanged {
						s.Names = astutil.Squeeze(s.Names)
						s.Values = astutil.Squeeze(s.Values)
						if len(s.Names) == 0 {
							declChanged = true
							d.Specs[j] = nil
						}
					}
				}
			}
			if declChanged {
				d.Specs = astutil.Squeeze(d.Specs)
				if len(d.Specs) == 0 {
					fileChanged = true
					file.Decls[i] = nil
				}
			}
		}
	}
	if fileChanged {
		file.Decls = astutil.Squeeze(file.Decls)
	}
	file.Imports = astutil.Squeeze(file.Imports)
	file.Comments = nil
}

// Options controls build process behavior.
type Options struct {
	Verbose        bool
	Quiet          bool
	Watch          bool
	CreateMapFile  bool
	MapToLocalDisk bool
	Minify         bool
	Color          bool
	BuildTags      []string
	TestedPackage  string
	NoCache        bool
}

// PrintError message to the terminal.
func (o *Options) PrintError(format string, a ...interface{}) {
	if o.Color {
		format = "\x1B[31m" + format + "\x1B[39m"
	}
	fmt.Fprintf(os.Stderr, format, a...)
}

// PrintSuccess message to the terminal.
func (o *Options) PrintSuccess(format string, a ...interface{}) {
	if o.Color {
		format = "\x1B[32m" + format + "\x1B[39m"
	}
	fmt.Fprintf(os.Stderr, format, a...)
}

// JSFile represents a *.inc.js file metadata and content.
type JSFile struct {
	Path    string // Full file path for the build context the file came from.
	ModTime time.Time
	Content []byte
}

// PackageData is an extension of go/build.Package with additional metadata
// GopherJS requires.
type PackageData struct {
	*build.Package
	JSFiles []JSFile
	// IsTest is true if the package is being built for running tests.
	IsTest     bool
	SrcModTime time.Time
	UpToDate   bool
	// If true, the package does not have a corresponding physical directory on disk.
	IsVirtual bool

	bctx *build.Context // The original build context this package came from.
}

func (p PackageData) String() string {
	return fmt.Sprintf("%s [is_test=%v]", p.ImportPath, p.IsTest)
}

// FileModTime returns the most recent modification time of the package's source
// files. This includes all .go and .inc.js that would be included in the build,
// but excludes any dependencies.
func (p PackageData) FileModTime() time.Time {
	newest := time.Time{}
	for _, file := range p.JSFiles {
		if file.ModTime.After(newest) {
			newest = file.ModTime
		}
	}

	// Unfortunately, build.Context methods don't allow us to Stat and individual
	// file, only to enumerate a directory. So we first get mtimes for all files
	// in the package directory, and then pick the newest for the relevant GoFiles.
	mtimes := map[string]time.Time{}
	files, err := buildutil.ReadDir(p.bctx, p.Dir)
	if err != nil {
		log.Errorf("Failed to enumerate files in the %q in context %v: %s. Assuming time.Now().", p.Dir, p.bctx, err)
		return time.Now()
	}
	for _, file := range files {
		mtimes[file.Name()] = file.ModTime()
	}

	for _, file := range p.GoFiles {
		t, ok := mtimes[file]
		if !ok {
			log.Errorf("No mtime found for source file %q of package %q, assuming time.Now().", file, p.Name)
			return time.Now()
		}
		if t.After(newest) {
			newest = t
		}
	}
	return newest
}

// InternalBuildContext returns the build context that produced the package.
//
// WARNING: This function is a part of internal API and will be removed in
// future.
func (p *PackageData) InternalBuildContext() *build.Context {
	return p.bctx
}

// TestPackage returns a variant of the package with "internal" tests.
func (p *PackageData) TestPackage() *PackageData {
	return &PackageData{
		Package: &build.Package{
			Name:            p.Name,
			ImportPath:      p.ImportPath,
			Dir:             p.Dir,
			GoFiles:         append(p.GoFiles, p.TestGoFiles...),
			Imports:         append(p.Imports, p.TestImports...),
			EmbedPatternPos: joinEmbedPatternPos(p.EmbedPatternPos, p.TestEmbedPatternPos),
		},
		IsTest:  true,
		JSFiles: p.JSFiles,
		bctx:    p.bctx,
	}
}

// XTestPackage returns a variant of the package with "external" tests.
func (p *PackageData) XTestPackage() *PackageData {
	return &PackageData{
		Package: &build.Package{
			Name:            p.Name + "_test",
			ImportPath:      p.ImportPath + "_test",
			Dir:             p.Dir,
			GoFiles:         p.XTestGoFiles,
			Imports:         p.XTestImports,
			EmbedPatternPos: p.XTestEmbedPatternPos,
		},
		IsTest: true,
		bctx:   p.bctx,
	}
}

// InstallPath returns the path where "gopherjs install" command should place the
// generated output.
func (p *PackageData) InstallPath() string {
	if p.IsCommand() {
		name := filepath.Base(p.ImportPath) + ".js"
		// For executable packages, mimic go tool behavior if possible.
		if gobin := os.Getenv("GOBIN"); gobin != "" {
			return filepath.Join(gobin, name)
		} else if gopath := os.Getenv("GOPATH"); gopath != "" {
			return filepath.Join(gopath, "bin", name)
		} else if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "go", "bin", name)
		}
	}
	return p.PkgObj
}

// Session manages internal state GopherJS requires to perform a build.
//
// This is the main interface to GopherJS build system. Session lifetime is
// roughly equivalent to a single GopherJS tool invocation.
type Session struct {
	options    *Options
	xctx       XContext
	buildCache cache.BuildCache

	// Binary archives produced during the current session and assumed to be
	// up to date with input sources and dependencies. In the -w ("watch") mode
	// must be cleared upon entering watching.
	UpToDateArchives map[string]*compiler.Archive
	Types            map[string]*types.Package
	Watcher          *fsnotify.Watcher
}

// NewSession creates a new GopherJS build session.
func NewSession(options *Options) (*Session, error) {
	options.Verbose = options.Verbose || options.Watch

	s := &Session{
		options:          options,
		UpToDateArchives: make(map[string]*compiler.Archive),
	}
	s.xctx = NewBuildContext(s.InstallSuffix(), s.options.BuildTags)
	env := s.xctx.Env()

	// Go distribution version check.
	if err := compiler.CheckGoVersion(env.GOROOT); err != nil {
		return nil, err
	}

	s.buildCache = cache.BuildCache{
		GOOS:          env.GOOS,
		GOARCH:        env.GOARCH,
		GOROOT:        env.GOROOT,
		GOPATH:        env.GOPATH,
		BuildTags:     append([]string{}, env.BuildTags...),
		Minify:        options.Minify,
		TestedPackage: options.TestedPackage,
	}
	s.Types = make(map[string]*types.Package)
	if options.Watch {
		if out, err := exec.Command("ulimit", "-n").Output(); err == nil {
			if n, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil && n < 1024 {
				fmt.Printf("Warning: The maximum number of open file descriptors is very low (%d). Change it with 'ulimit -n 8192'.\n", n)
			}
		}

		var err error
		s.Watcher, err = fsnotify.NewWatcher()
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

// XContext returns the session's build context.
func (s *Session) XContext() XContext { return s.xctx }

// InstallSuffix returns the suffix added to the generated output file.
func (s *Session) InstallSuffix() string {
	if s.options.Minify {
		return "min"
	}
	return ""
}

// GoRelease returns Go release version this session is building with.
func (s *Session) GoRelease() string {
	return compiler.GoRelease(s.xctx.Env().GOROOT)
}

// BuildFiles passed to the GopherJS tool as if they were a package.
//
// A ephemeral package will be created with only the provided files. This
// function is intended for use with, for example, `gopherjs run main.go`.
func (s *Session) BuildFiles(filenames []string, pkgObj string, cwd string) error {
	if len(filenames) == 0 {
		return fmt.Errorf("no input sources are provided")
	}

	normalizedDir := func(filename string) string {
		d := filepath.Dir(filename)
		if !filepath.IsAbs(d) {
			d = filepath.Join(cwd, d)
		}
		return filepath.Clean(d)
	}

	// Ensure all source files are in the same directory.
	dirSet := map[string]bool{}
	for _, file := range filenames {
		dirSet[normalizedDir(file)] = true
	}
	dirList := []string{}
	for dir := range dirSet {
		dirList = append(dirList, dir)
	}
	sort.Strings(dirList)
	if len(dirList) != 1 {
		return fmt.Errorf("named files must all be in one directory; have: %v", strings.Join(dirList, ", "))
	}

	root := dirList[0]
	ctx := build.Default
	ctx.UseAllFiles = true
	ctx.ReadDir = func(dir string) ([]fs.FileInfo, error) {
		n := len(filenames)
		infos := make([]fs.FileInfo, n)
		for i := 0; i < n; i++ {
			info, err := os.Stat(filenames[i])
			if err != nil {
				return nil, err
			}
			infos[i] = info
		}
		return infos, nil
	}
	p, err := ctx.Import(".", root, 0)
	if err != nil {
		return err
	}
	p.Name = "main"
	p.ImportPath = "main"

	pkg := &PackageData{
		Package: p,
		// This ephemeral package doesn't have a unique import path to be used as a
		// build cache key, so we never cache it.
		SrcModTime: time.Now().Add(time.Hour),
		bctx:       &goCtx(s.xctx.Env()).bctx,
	}

	for _, file := range filenames {
		if !strings.HasSuffix(file, ".inc.js") {
			continue
		}

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}
		info, err := os.Stat(file)
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", file, err)
		}
		pkg.JSFiles = append(pkg.JSFiles, JSFile{
			Path:    filepath.Join(pkg.Dir, filepath.Base(file)),
			ModTime: info.ModTime(),
			Content: content,
		})
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

// BuildImportPath loads and compiles package with the given import path.
//
// Relative paths are interpreted relative to the current working dir.
func (s *Session) BuildImportPath(path string) (*compiler.Archive, error) {
	_, archive, err := s.buildImportPathWithSrcDir(path, "")
	return archive, err
}

// buildImportPathWithSrcDir builds the package specified by the import path.
//
// Relative import paths are interpreted relative to the passed srcDir. If
// srcDir is empty, current working directory is assumed.
func (s *Session) buildImportPathWithSrcDir(path string, srcDir string) (*PackageData, *compiler.Archive, error) {
	pkg, err := s.xctx.Import(path, srcDir, 0)
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

// BuildPackage compiles an already loaded package.
func (s *Session) BuildPackage(pkg *PackageData) (*compiler.Archive, error) {
	if archive, ok := s.UpToDateArchives[pkg.ImportPath]; ok {
		return archive, nil
	}

	var fileInfo os.FileInfo
	gopherjsBinary, err := os.Executable()
	if err == nil {
		fileInfo, err = os.Stat(gopherjsBinary)
		if err == nil && fileInfo.ModTime().After(pkg.SrcModTime) {
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
		importedPkg, _, err := s.buildImportPathWithSrcDir(importedPkgPath, pkg.Dir)
		if err != nil {
			return nil, err
		}

		impModTime := importedPkg.SrcModTime
		if impModTime.After(pkg.SrcModTime) {
			pkg.SrcModTime = impModTime
		}
	}

	if pkg.FileModTime().After(pkg.SrcModTime) {
		pkg.SrcModTime = pkg.FileModTime()
	}

	if !s.options.NoCache {
		archive := s.buildCache.LoadArchive(pkg.ImportPath)
		if archive != nil && !pkg.SrcModTime.After(archive.BuildTime) {
			if err := archive.RegisterTypes(s.Types); err != nil {
				panic(fmt.Errorf("failed to load type information from %v: %w", archive, err))
			}
			s.UpToDateArchives[pkg.ImportPath] = archive
			// Existing archive is up to date, no need to build it from scratch.
			return archive, nil
		}
	}

	// Existing archive is out of date or doesn't exist, let's build the package.
	fileSet := token.NewFileSet()
	files, overlayJsFiles, err := parseAndAugment(s.xctx, pkg, pkg.IsTest, fileSet)
	if err != nil {
		return nil, err
	}
	embed, err := embedFiles(pkg, fileSet, files)
	if err != nil {
		return nil, err
	}
	if embed != nil {
		files = append(files, embed)
	}

	importContext := &compiler.ImportContext{
		Packages: s.Types,
		Import:   s.ImportResolverFor(pkg),
	}
	archive, err := compiler.Compile(pkg.ImportPath, files, fileSet, importContext, s.options.Minify)
	if err != nil {
		return nil, err
	}

	for _, jsFile := range append(pkg.JSFiles, overlayJsFiles...) {
		archive.IncJSCode = append(archive.IncJSCode, []byte("\t(function() {\n")...)
		archive.IncJSCode = append(archive.IncJSCode, jsFile.Content...)
		archive.IncJSCode = append(archive.IncJSCode, []byte("\n\t}).call($global);\n")...)
	}

	if s.options.Verbose {
		fmt.Println(pkg.ImportPath)
	}

	s.buildCache.StoreArchive(archive)
	s.UpToDateArchives[pkg.ImportPath] = archive

	return archive, nil
}

// ImportResolverFor returns a function which returns a compiled package archive
// given an import path.
func (s *Session) ImportResolverFor(pkg *PackageData) func(string) (*compiler.Archive, error) {
	return func(path string) (*compiler.Archive, error) {
		if archive, ok := s.UpToDateArchives[path]; ok {
			return archive, nil
		}
		_, archive, err := s.buildImportPathWithSrcDir(path, pkg.Dir)
		return archive, err
	}
}

// SourceMappingCallback returns a call back for compiler.SourceMapFilter
// configured for the current build session.
func (s *Session) SourceMappingCallback(m *sourcemap.Map) func(generatedLine, generatedColumn int, originalPos token.Position) {
	return NewMappingCallback(m, s.xctx.Env().GOROOT, s.xctx.Env().GOPATH, s.options.MapToLocalDisk)
}

// WriteCommandPackage writes the final JavaScript output file at pkgObj path.
func (s *Session) WriteCommandPackage(archive *compiler.Archive, pkgObj string) error {
	if err := os.MkdirAll(filepath.Dir(pkgObj), 0o777); err != nil {
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

		sourceMapFilter.MappingCallback = s.SourceMappingCallback(m)
	}

	deps, err := compiler.ImportDependencies(archive, func(path string) (*compiler.Archive, error) {
		if archive, ok := s.UpToDateArchives[path]; ok {
			return archive, nil
		}
		_, archive, err := s.buildImportPathWithSrcDir(path, "")
		return archive, err
	})
	if err != nil {
		return err
	}
	return compiler.WriteProgramCode(deps, sourceMapFilter, s.GoRelease())
}

// NewMappingCallback creates a new callback for source map generation.
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

// WaitForChange watches file system events and returns if either when one of
// the source files is modified.
func (s *Session) WaitForChange() {
	// Will need to re-validate up-to-dateness of all archives, so flush them from
	// memory.
	s.UpToDateArchives = map[string]*compiler.Archive{}
	s.Types = map[string]*types.Package{}

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
