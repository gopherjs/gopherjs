package build

import (
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/gopherjs/gopherjs/compiler"
)

// dirCache remembers where non-GOROOT packages live on disk.
//
// In module mode go/build runs `go list` once per imported package to find its
// directory, which dominates load time for large programs. dirCache instead
// runs `go list -deps` on a miss and records the location of every package in
// the dependency tree, so the rest of the tree is resolved without spawning the
// go command again.
//
// go/build runs `go list` in build.Context.Dir regardless of the importing
// package, so a package's location depends only on its import path for a given
// context. A dirCache must therefore not be shared between contexts.
type dirCache struct {
	mu      sync.Mutex
	entries map[string]*dirEntry // nil means looked up but not found
}

type dirEntry struct {
	importPath string
	dir        string
	root       string
}

func newDirCache() *dirCache {
	return &dirCache{entries: map[string]*dirEntry{}}
}

// importPackage is equivalent to bctx.Import, but resolves package directories
// from the cache when go/build would otherwise have run `go list`.
func (dc *dirCache) importPackage(bctx build.Context, importPath, srcDir string, mode build.ImportMode) (*build.Package, error) {
	if dc == nil || !resolvesWithGoList(bctx, importPath, srcDir, mode) {
		return bctx.Import(importPath, srcDir, mode)
	}
	entry := dc.lookup(bctx, importPath)
	if entry == nil {
		return bctx.Import(importPath, srcDir, mode)
	}

	pkg, err := bctx.ImportDir(entry.dir, mode)
	if pkg == nil {
		return pkg, err
	}
	setListedLocation(bctx, pkg, *entry)
	return pkg, err
}

// resolvesWithGoList reports whether bctx.Import would locate importPath by
// running `go list`. It mirrors the conditions in go/build's importGo, and errs
// on the side of returning false.
func resolvesWithGoList(bctx build.Context, importPath, srcDir string, mode build.ImportMode) bool {
	if os.Getenv("GOPHERJS_NO_DIRCACHE") != "" {
		return false
	}
	if mode&(build.AllowBinary|build.IgnoreVendor) != 0 {
		return false
	}
	if bctx.JoinPath != nil || bctx.SplitPathList != nil || bctx.IsAbsPath != nil || bctx.IsDir != nil ||
		bctx.HasSubdir != nil || bctx.ReadDir != nil || bctx.OpenFile != nil {
		return false
	}
	// versionhack makes these the go/build defaults that importGo compares against.
	if !slices.Equal(bctx.ToolTags, build.Default.ToolTags) ||
		!slices.Equal(bctx.ReleaseTags, build.Default.ReleaseTags[:compiler.GoVersion]) {
		return false
	}
	if bctx.Compiler != "gc" || bctx.GOROOT == "" {
		return false
	}
	// "auto" needs a go.mod lookup to decide; leave that to go/build.
	if gm := os.Getenv("GO111MODULE"); gm == "off" || gm == "auto" {
		return false
	}
	if importPath == "" || build.IsLocalImport(importPath) || strings.HasPrefix(importPath, "/") {
		return false
	}
	if srcDir != "" {
		if !filepath.IsAbs(srcDir) && bctx.Dir != "" {
			return false
		}
		absSrcDir, err := filepath.Abs(srcDir)
		if err != nil || isSubdir(filepath.Join(bctx.GOROOT, "src"), absSrcDir) {
			return false
		}
	}
	if fi, err := os.Stat(filepath.Join(bctx.GOROOT, "src", importPath)); err == nil && fi.IsDir() {
		return false
	}
	return true
}

// isSubdir reports whether dir is root or inside it, also comparing the paths
// with symlinks resolved, as go/build's hasSubdir does.
func isSubdir(root, dir string) bool {
	inside := func(root, dir string) bool {
		rel, err := filepath.Rel(root, dir)
		return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
	}
	if inside(root, dir) {
		return true
	}
	realRoot, err1 := filepath.EvalSymlinks(root)
	realDir, err2 := filepath.EvalSymlinks(dir)
	if err1 != nil || err2 != nil {
		return false
	}
	return inside(realRoot, dir) || inside(root, realDir) || inside(realRoot, realDir)
}

// lookup returns the location of importPath, listing its dependency tree on
// the first miss. The lock is held while `go list` runs so that concurrent
// misses don't list the same tree twice.
func (dc *dirCache) lookup(bctx build.Context, importPath string) *dirEntry {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	if e, ok := dc.entries[importPath]; ok {
		return e
	}
	dc.entries[importPath] = nil
	if err := dc.listDeps(bctx, importPath); err != nil {
		return nil
	}
	return dc.entries[importPath]
}

// listDeps runs `go list -deps` the same way go/build's importGo runs
// `go list`, and records every non-GOROOT package that has a directory.
// Packages go list couldn't locate are left out, so go/build reports the error.
func (dc *dirCache) listDeps(bctx build.Context, importPath string) error {
	cmd := exec.Command(filepath.Join(bctx.GOROOT, "bin", "go"), "list", "-e", "-deps",
		"-compiler="+bctx.Compiler,
		"-tags="+strings.Join(bctx.BuildTags, ","),
		"-installsuffix="+bctx.InstallSuffix,
		"-f={{if .Dir}}{{.ImportPath}}\t{{.Dir}}\t{{.Root}}\t{{.Goroot}}{{end}}",
		"--", importPath)
	if bctx.Dir != "" {
		cmd.Dir = bctx.Dir
	}
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cgo := "0"
	if bctx.CgoEnabled {
		cgo = "1"
	}
	cmd.Env = append(cmd.Environ(),
		"GOOS="+bctx.GOOS,
		"GOARCH="+bctx.GOARCH,
		"GOROOT="+bctx.GOROOT,
		"GOPATH="+bctx.GOPATH,
		"CGO_ENABLED="+cgo,
	)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go list -deps %s: %w\n%s", importPath, err, stderr.String())
	}
	for _, line := range strings.Split(stdout.String(), "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 4 || f[3] == "true" {
			continue
		}
		if dc.entries[f[0]] == nil {
			dc.entries[f[0]] = &dirEntry{importPath: f[0], dir: f[1], root: f[2]}
		}
	}
	return nil
}

// setListedLocation sets the fields that bctx.Import derives from the `go list`
// result, replacing the ones ImportDir derived from the directory alone.
func setListedLocation(bctx build.Context, pkg *build.Package, e dirEntry) {
	pkg.ImportPath = e.importPath
	pkg.Root = e.root
	pkg.Goroot = false
	pkg.ConflictDir = ""
	if e.root == "" {
		pkg.SrcRoot, pkg.PkgRoot, pkg.BinDir, pkg.PkgTargetRoot, pkg.PkgObj = "", "", "", "", ""
		return
	}
	suffix := ""
	if bctx.InstallSuffix != "" {
		suffix = "_" + bctx.InstallSuffix
	}
	pkgTargetRoot := "pkg/" + bctx.GOOS + "_" + bctx.GOARCH + suffix
	pkg.SrcRoot = filepath.Join(e.root, "src")
	pkg.PkgRoot = filepath.Join(e.root, "pkg")
	pkg.BinDir = filepath.Join(e.root, "bin")
	pkg.PkgTargetRoot = filepath.Join(e.root, pkgTargetRoot)
	pkg.PkgObj = filepath.Join(e.root, pkgTargetRoot+"/"+e.importPath+".a")
}
