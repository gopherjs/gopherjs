package build

import (
	"fmt"
	"go/build"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gopherjs/gopherjs/compiler"
	"github.com/kisielk/gotool"
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
	Match(patterns []string) []string
}

// simpleCtx is a wrapper around go/build.Context with support for GopherJS-specific
// features.
type simpleCtx struct {
	bctx      build.Context
	isVirtual bool // Imported packages don't have a physical directory on disk.
}

// Import implements XContext.Import().
func (sc simpleCtx) Import(importPath string, srcDir string, mode build.ImportMode) (*PackageData, error) {
	bctx, mode := sc.applyPackageTweaks(importPath, mode)
	pkg, err := bctx.Import(importPath, srcDir, mode)
	if err != nil {
		return nil, err
	}
	jsFiles, err := jsFilesFromDir(&sc.bctx, pkg.Dir)
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate .inc.js files in %s: %w", pkg.Dir, err)
	}
	pkg.PkgObj = sc.rewritePkgObj(pkg.PkgObj)
	return &PackageData{
		Package:   pkg,
		IsVirtual: sc.isVirtual,
		JSFiles:   jsFiles,
		bctx:      &sc.bctx,
	}, nil
}

// Match implements XContext.Match.
func (sc simpleCtx) Match(patterns []string) []string {
	// TODO(nevkontakte): The gotool library prints warnings directly to stderr
	// when it matched no packages. This may be misleading with chained contexts
	// when a package is only found in the secondary context. Perhaps we could
	// replace gotool package with golang.org/x/tools/go/buildutil.ExpandPatterns()
	// at the cost of a slightly more limited pattern support compared to go tool?
	tool := gotool.Context{BuildContext: sc.bctx}
	return tool.ImportPaths(patterns)
}

func (sc simpleCtx) GOOS() string { return sc.bctx.GOOS }

// applyPackageTweaks makes several package-specific adjustments to package importing.
//
// Ideally this method would not be necessary, but currently several packages
// require special handing in order to be compatible with GopherJS. This method
// returns a copy of the build context, keeping the original one intact.
func (sc simpleCtx) applyPackageTweaks(importPath string, mode build.ImportMode) (build.Context, build.ImportMode) {
	bctx := sc.bctx
	switch importPath {
	case "syscall":
		// syscall needs to use a typical GOARCH like amd64 to pick up definitions for _Socklen, BpfInsn, IFNAMSIZ, Timeval, BpfStat, SYS_FCNTL, Flock_t, etc.
		bctx.GOARCH = build.Default.GOARCH
		bctx.InstallSuffix += build.Default.GOARCH
	case "syscall/js":
		// There are no buildable files in this package, but we need to use files in the virtual directory.
		mode |= build.FindOnly
	case "crypto/x509", "os/user":
		// These stdlib packages have cgo and non-cgo versions (via build tags); we want the latter.
		bctx.CgoEnabled = false
	case "github.com/gopherjs/gopherjs/js", "github.com/gopherjs/gopherjs/nosync":
		// These packages are already embedded via gopherjspkg.FS virtual filesystem (which can be
		// safely vendored). Don't try to use vendor directory to resolve them.
		mode |= build.IgnoreVendor
	}

	return bctx, mode
}

func (sc simpleCtx) rewritePkgObj(orig string) string {
	if orig == "" {
		return orig
	}

	goroot := mustAbs(sc.bctx.GOROOT)
	gopath := mustAbs(sc.bctx.GOPATH)
	orig = mustAbs(orig)

	if strings.HasPrefix(orig, filepath.Join(gopath, "pkg", "mod")) {
		// Go toolchain makes sources under GOPATH/pkg/mod readonly, so we can't
		// store our artifacts there.
		return cachedPath(orig)
	}

	allowed := []string{goroot, gopath}
	for _, prefix := range allowed {
		if strings.HasPrefix(orig, prefix) {
			// Traditional GOPATH-style locations for build artifacts are ok to use.
			return orig
		}
	}

	// Everything else also goes into the cache just in case.
	return cachedPath(orig)
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
func (cc chainedCtx) Match(patterns []string) []string {
	seen := map[string]bool{}
	matches := []string{}
	for _, m := range append(cc.primary.Match(patterns), cc.secondary.Match(patterns)...) {
		if seen[m] {
			continue
		}
		seen[m] = true
		matches = append(matches, m)
	}
	sort.Strings(matches)
	return matches
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
