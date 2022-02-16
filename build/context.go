package build

import (
	"fmt"
	"go/build"
	"go/token"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gopherjs/gopherjs/compiler"
	"golang.org/x/tools/go/buildutil"
)

// XContext is an extension of go/build.Context with GopherJS-specifc features.
//
// It abstracts away several different sources GopherJS can load its packages
// from, with a minimal API.
type XContext interface {
	// Import returns details about the Go package named by the importPath,
	// interpreting local import paths relative to the srcDir directory.
	Import(path string, srcDir string, mode build.ImportMode) (*PackageData, error)

	// GOOS returns GOOS value the underlying build.Context is using.
	// This will become obsolete after https://github.com/gopherjs/gopherjs/issues/693.
	GOOS() string

	// Match explans build patterns into a set of matching import paths (see go help packages).
	Match(patterns []string) ([]string, error)
}

// simpleCtx is a wrapper around go/build.Context with support for GopherJS-specific
// features.
type simpleCtx struct {
	bctx      build.Context
	isVirtual bool // Imported packages don't have a physical directory on disk.
}

// Import implements XContext.Import().
func (sc simpleCtx) Import(importPath string, srcDir string, mode build.ImportMode) (*PackageData, error) {
	bctx, mode := sc.applyPreloadTweaks(importPath, mode)
	pkg, err := bctx.Import(importPath, srcDir, mode)
	if err != nil {
		return nil, err
	}
	jsFiles, err := jsFilesFromDir(&sc.bctx, pkg.Dir)
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate .inc.js files in %s: %w", pkg.Dir, err)
	}
	if !path.IsAbs(pkg.Dir) {
		pkg.Dir = mustAbs(pkg.Dir)
	}
	pkg = sc.applyPostloadTweaks(pkg)

	if len(pkg.CgoFiles) > 0 {
		return nil, &ImportCError{pkg.ImportPath}
	}

	return &PackageData{
		Package:   pkg,
		IsVirtual: sc.isVirtual,
		JSFiles:   jsFiles,
		bctx:      &sc.bctx,
	}, nil
}

// Match implements XContext.Match.
func (sc simpleCtx) Match(patterns []string) ([]string, error) {
	if sc.isVirtual {
		// We can't use go tool to enumerate packages in a virtual file system,
		// so we fall back onto a simpler implementation provided by the buildutil
		// package. It doesn't support all valid patterns, but should be good enough.
		//
		// Note: this code path will become unnecessary after
		// https://github.com/gopherjs/gopherjs/issues/1021 is implemented.
		args := []string{}
		for _, p := range patterns {
			switch p {
			case "all":
				args = append(args, "...")
			case "std", "main", "cmd":
				// These patterns are not supported by buildutil.ExpandPatterns(),
				// but they would be matched by the real context correctly, so skip them.
			default:
				args = append(args, p)
			}
		}
		matches := []string{}
		for importPath := range buildutil.ExpandPatterns(&sc.bctx, args) {
			if importPath[0] == '.' {
				p, err := sc.Import(importPath, ".", build.FindOnly)
				// Resolve relative patterns into canonical import paths.
				if err != nil {
					continue
				}
				importPath = p.ImportPath
			}
			matches = append(matches, importPath)
		}
		sort.Strings(matches)
		return matches, nil
	}

	args := append([]string{
		"-e", "-compiler=gc",
		"-tags=" + strings.Join(sc.bctx.BuildTags, ","),
		"-installsuffix=" + sc.bctx.InstallSuffix,
		"-f={{.ImportPath}}",
		"--",
	}, patterns...)

	out, err := sc.gotool("list", args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list packages on FS: %w", err)
	}
	matches := strings.Split(strings.TrimSpace(out), "\n")
	sort.Strings(matches)
	return matches, nil
}

func (sc simpleCtx) GOOS() string { return sc.bctx.GOOS }

// gotool executes the go tool set up for the build context and returns standard output.
func (sc simpleCtx) gotool(subcommand string, args ...string) (string, error) {
	if sc.isVirtual {
		panic(fmt.Errorf("can't use go tool with a virtual build context"))
	}
	args = append([]string{subcommand}, args...)
	cmd := exec.Command("go", args...)

	if sc.bctx.Dir != "" {
		cmd.Dir = sc.bctx.Dir
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	cgo := "0"
	if sc.bctx.CgoEnabled {
		cgo = "1"
	}
	cmd.Env = append(os.Environ(),
		"GOOS="+sc.bctx.GOOS,
		"GOARCH="+sc.bctx.GOARCH,
		"GOROOT="+sc.bctx.GOROOT,
		"GOPATH="+sc.bctx.GOPATH,
		"CGO_ENABLED="+cgo,
	)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go tool error: %v: %w\n%s", cmd, err, stderr.String())
	}
	return stdout.String(), nil
}

// applyPreloadTweaks makes several package-specific adjustments to package importing.
//
// Ideally this method would not be necessary, but currently several packages
// require special handing in order to be compatible with GopherJS. This method
// returns a copy of the build context, keeping the original one intact.
func (sc simpleCtx) applyPreloadTweaks(importPath string, mode build.ImportMode) (build.Context, build.ImportMode) {
	bctx := sc.bctx
	switch importPath {
	case "syscall":
		// syscall needs to use a typical GOARCH like amd64 to pick up definitions
		// for _Socklen, BpfInsn, IFNAMSIZ, Timeval, BpfStat, SYS_FCNTL, Flock_t,
		// etc.
		bctx.GOARCH = build.Default.GOARCH
		bctx.InstallSuffix += build.Default.GOARCH
	case "syscall/js":
		if !sc.isVirtual {
			// There are no buildable files in this package upstream, but we need to
			// use files in the virtual directory.
			mode |= build.FindOnly
		}
	case "crypto/x509", "os/user":
		// These stdlib packages have cgo and non-cgo versions (via build tags); we
		// want the latter.
		bctx.CgoEnabled = false
	case "github.com/gopherjs/gopherjs/js", "github.com/gopherjs/gopherjs/nosync":
		// These packages are already embedded via gopherjspkg.FS virtual filesystem
		// (which can be safely vendored). Don't try to use vendor directory to
		// resolve them.
		mode |= build.IgnoreVendor
	}

	return bctx, mode
}

// applyPostloadTweaks makes adjustments to the contents of the loaded package.
//
// Some of the standard library packages require additional tweaks that are not
// covered by our augmentation logic, for example excluding or including
// particular source files. This method ensures that all such tweaks are applied
// before the package is returned to the caller.
func (sc simpleCtx) applyPostloadTweaks(pkg *build.Package) *build.Package {
	if sc.isVirtual {
		// GopherJS overlay package sources don't need tweaks to their content,
		// since we already control them directly.
		return pkg
	}
	switch pkg.ImportPath {
	case "os":
		pkg.GoFiles = excludeExecutable(pkg.GoFiles) // Need to exclude executable implementation files, because some of them contain package scope variables that perform (indirectly) syscalls on init.
		// Prefer the dirent_${GOOS}.go version, to make the build pass on both linux
		// and darwin.
		// In the long term, our builds should produce the same output regardless
		// of the host OS: https://github.com/gopherjs/gopherjs/issues/693.
		pkg.GoFiles = exclude(pkg.GoFiles, "dirent_js.go")
	case "runtime":
		pkg.GoFiles = []string{} // Package sources are completely replaced in natives.
	case "runtime/internal/sys":
		pkg.GoFiles = []string{fmt.Sprintf("zgoos_%s.go", sc.GOOS()), "zversion.go"}
	case "runtime/pprof":
		pkg.GoFiles = nil
	case "internal/poll":
		pkg.GoFiles = exclude(pkg.GoFiles, "fd_poll_runtime.go")
	case "sync":
		// GopherJS completely replaces sync.Pool implementation with a simpler one,
		// since it always executes in a single-threaded environment.
		pkg.GoFiles = exclude(pkg.GoFiles, "pool.go")
	case "crypto/rand":
		pkg.GoFiles = []string{"rand.go", "util.go"}
		pkg.TestGoFiles = exclude(pkg.TestGoFiles, "rand_linux_test.go") // Don't want linux-specific tests (since linux-specific package files are excluded too).
	case "crypto/x509":
		// GopherJS doesn't support loading OS root certificates regardless of the
		// OS. The substitution below allows to avoid build dependency on Mac OS
		// implementation, which won't be used anyway.
		//
		// Just like above, https://github.com/gopherjs/gopherjs/issues/693 is
		// probably the best long-term option.
		pkg.GoFiles = include(
			exclude(pkg.GoFiles, fmt.Sprintf("root_%s.go", sc.GOOS())),
			"root_unix.go", "root_js.go")
	case "syscall/js":
		// Reuse upstream tests to ensure conformance, but completely replace
		// implementation.
		pkg.GoFiles = []string{}
		pkg.XTestGoFiles = append(pkg.XTestGoFiles, "js_test.go")
	}

	pkg.Imports, pkg.ImportPos = updateImports(pkg.GoFiles, pkg.ImportPos)
	pkg.TestImports, pkg.TestImportPos = updateImports(pkg.TestGoFiles, pkg.TestImportPos)
	pkg.XTestImports, pkg.XTestImportPos = updateImports(pkg.XTestGoFiles, pkg.XTestImportPos)

	return pkg
}

var defaultBuildTags = []string{
	"netgo",            // See https://godoc.org/net#hdr-Name_Resolution.
	"purego",           // See https://golang.org/issues/23172.
	"math_big_pure_go", // Use pure Go version of math/big.
}

// embeddedCtx creates simpleCtx that imports from a virtual FS embedded into
// the GopherJS compiler.
func embeddedCtx(embedded http.FileSystem, installSuffix string, buildTags []string) *simpleCtx {
	fs := &vfs{embedded}
	ec := goCtx(installSuffix, buildTags)
	ec.bctx.GOPATH = ""

	// Path functions must behave unix-like to work with the VFS.
	ec.bctx.JoinPath = path.Join
	ec.bctx.SplitPathList = splitPathList
	ec.bctx.IsAbsPath = path.IsAbs
	ec.bctx.HasSubdir = hasSubdir

	// Substitute real FS with the embedded one.
	ec.bctx.IsDir = fs.IsDir
	ec.bctx.ReadDir = fs.ReadDir
	ec.bctx.OpenFile = fs.OpenFile
	ec.isVirtual = true
	return ec
}

// goCtx creates simpleCtx that imports from the real file system GOROOT, GOPATH
// or Go Modules.
func goCtx(installSuffix string, buildTags []string) *simpleCtx {
	gc := simpleCtx{
		bctx: build.Context{
			GOROOT:        DefaultGOROOT,
			GOPATH:        build.Default.GOPATH,
			GOOS:          build.Default.GOOS,
			GOARCH:        "js",
			InstallSuffix: installSuffix,
			Compiler:      "gc",
			BuildTags:     append(buildTags, defaultBuildTags...),
			CgoEnabled:    true, // detect `import "C"` to throw proper error

			// go/build supports modules, but only when no FS access functions are
			// overridden and when provided ReleaseTags match those of the default
			// context (matching Go compiler's version).
			// This limitation stems from the fact that it will invoke the Go tool
			// which can only see files on the real FS and will assume release tags
			// based on the Go tool's version.
			//
			// See also comments to the versionhack package.
			ReleaseTags: build.Default.ReleaseTags[:compiler.GoVersion],
		},
	}
	return &gc
}

// chainedCtx combines two build contexts. Secondary context acts as a fallback
// when a package is not found in the primary, and is ignored otherwise.
//
// This allows GopherJS to load its core "js" and "nosync" packages from the
// embedded VFS whenever user's code doesn't directly depend on them, but
// augmented stdlib does.
type chainedCtx struct {
	primary   XContext
	secondary XContext
}

// Import implements buildCtx.Import().
func (cc chainedCtx) Import(importPath string, srcDir string, mode build.ImportMode) (*PackageData, error) {
	pkg, err := cc.primary.Import(importPath, srcDir, mode)
	if err == nil {
		return pkg, nil
	} else if IsPkgNotFound(err) {
		return cc.secondary.Import(importPath, srcDir, mode)
	} else {
		return nil, err
	}
}

func (cc chainedCtx) GOOS() string { return cc.primary.GOOS() }

// Match implements XContext.Match().
//
// Packages from both contexts are included and returned as a deduplicated
// sorted list.
func (cc chainedCtx) Match(patterns []string) ([]string, error) {
	m1, err := cc.primary.Match(patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to list packages in the primary context: %s", err)
	}
	m2, err := cc.secondary.Match(patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to list packages in the secondary context: %s", err)
	}

	seen := map[string]bool{}
	matches := []string{}
	for _, m := range append(m1, m2...) {
		if seen[m] {
			continue
		}
		seen[m] = true
		matches = append(matches, m)
	}
	sort.Strings(matches)
	return matches, nil
}

// IsPkgNotFound returns true if the error was caused by package not found.
//
// Unfortunately, go/build doesn't make use of typed errors, so we have to
// rely on the error message.
func IsPkgNotFound(err error) bool {
	return err != nil &&
		(strings.Contains(err.Error(), "cannot find package") || // Modules off.
			strings.Contains(err.Error(), "is not in GOROOT")) // Modules on.
}

// updateImports package's list of import paths to only those present in sources
// after post-load tweaks.
func updateImports(sources []string, importPos map[string][]token.Position) (newImports []string, newImportPos map[string][]token.Position) {
	if importPos == nil {
		// Short-circuit for tests when no imports are loaded.
		return nil, nil
	}
	sourceSet := map[string]bool{}
	for _, source := range sources {
		sourceSet[source] = true
	}

	newImportPos = map[string][]token.Position{}
	for importPath, positions := range importPos {
		for _, pos := range positions {
			if sourceSet[filepath.Base(pos.Filename)] {
				newImportPos[importPath] = append(newImportPos[importPath], pos)
			}
		}
	}

	for importPath := range newImportPos {
		newImports = append(newImports, importPath)
	}
	sort.Strings(newImports)
	return newImports, newImportPos
}
