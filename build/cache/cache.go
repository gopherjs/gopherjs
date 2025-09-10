// Package cache solves one of the hardest computer science problems in
// application to GopherJS compiler outputs.
package cache

import (
	"crypto/sha256"
	"fmt"
	"go/build"
	"go/types"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/gopherjs/gopherjs/compiler"
	log "github.com/sirupsen/logrus"
)

// cacheRoot is the base path for GopherJS's own build cache.
//
// It serves a similar function to the Go build cache, but is a lot more
// simplistic and therefore not compatible with Go. We use this cache directory
// to store build artifacts for packages loaded from a module, for which PkgObj
// provided by go/build points inside the module source tree, which can cause
// inconvenience with version control, etc.
var cacheRoot = func() string {
	path, err := os.UserCacheDir()
	if err == nil {
		return filepath.Join(path, "gopherjs", "build_cache")
	}

	return filepath.Join(build.Default.GOPATH, "pkg", "gopherjs_build_cache")
}()

// cachedPath returns a location inside the build cache for a given set of key
// strings. The set of keys must uniquely identify cacheable object. Prefer
// using more specific functions to ensure key consistency.
func cachedPath(keys ...string) string {
	key := path.Join(keys...)
	if key == "" {
		panic("CachedPath() must not be used with an empty string")
	}
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte(key)))
	return filepath.Join(cacheRoot, sum[0:2], sum)
}

// Clear the cache. This will remove *all* cached artifacts from *all* build
// configurations.
func Clear() error {
	return os.RemoveAll(cacheRoot)
}

// BuildCache manages build artifacts that are cached for incremental builds.
//
// Cache is designed to be non-durable: any store and load errors are swallowed
// and simply lead to a cache miss. The caller must be able to handle cache
// misses. Nil pointer to BuildCache is valid and simply disables caching.
//
// BuildCache struct fields represent build parameters which change invalidates
// the cache. For example, any artifacts that were cached for a minified build
// must not be reused for a non-minified build. GopherJS version change also
// invalidates the cache. It is callers responsibility to ensure that artifacts
// passed the StoreArchive function were generated with the same build
// parameters as the cache is configured.
//
// There is no upper limit for the total cache size. It can be cleared
// programmatically via the Clear() function, or the user can just delete the
// directory if it grows too big.
//
// TODO(nevkontakte): changes in the input sources or dependencies doesn't
// currently invalidate the cache. This is handled at the higher level by
// checking cached archive timestamp against loaded package modification time.
//
// TODO(nevkontakte): this cache could benefit from checksum integrity checks.
type BuildCache struct {
	GOOS      string
	GOARCH    string
	GOROOT    string
	GOPATH    string
	BuildTags []string
	Minify    bool
	// When building for tests, import path of the package being tested. The
	// package under test is built with *_test.go sources included, and since it
	// may be imported by other packages in the binary we can't reuse the "normal"
	// cache.
	TestedPackage string
}

func (bc BuildCache) String() string {
	return fmt.Sprintf("%#v", bc)
}

// StoreArchive compiled archive in the cache. Any error inside this method
// will cause the cache not to be persisted.
//
// The passed in buildTime is used to determine if the archive is out-of-date when reloaded.
// Typically it should be set to the srcModTime or time.Now().
func (bc *BuildCache) StoreArchive(a *compiler.Archive, buildTime time.Time) {
	if bc == nil {
		return // Caching is disabled.
	}
	path := cachedPath(bc.archiveKey(a.ImportPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		log.Warningf("Failed to create build cache directory: %v", err)
		return
	}
	// Write the archive in a temporary file first to avoid concurrency errors.
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path))
	if err != nil {
		log.Warningf("Failed to temporary build cache file: %v", err)
		return
	}
	defer f.Close()
	if err := compiler.WriteArchive(a, buildTime, f); err != nil {
		log.Warningf("Failed to write build cache archive %q: %v", a, err)
		// Make sure we don't leave a half-written archive behind.
		os.Remove(f.Name())
		return
	}
	f.Close()
	// Rename fully written file into its permanent name.
	if err := os.Rename(f.Name(), path); err != nil {
		log.Warningf("Failed to rename build cache archive to %q: %v", path, err)
	}
	log.Infof("Successfully stored build archive %q as %q.", a, path)
}

// LoadArchive returns a previously cached archive of the given package or nil
// if it wasn't previously stored.
//
// The returned archive would have been built with the same configuration as
// the build cache was.
//
// The imports map is used to resolve package dependencies and may modify the
// map to include the package from the read archive. See [gcexportdata.Read].
func (bc *BuildCache) LoadArchive(importPath string, srcModTime time.Time, imports map[string]*types.Package) *compiler.Archive {
	if bc == nil {
		return nil // Caching is disabled.
	}
	path := cachedPath(bc.archiveKey(importPath))
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Infof("No cached package archive for %q.", importPath)
		} else {
			log.Warningf("Failed to open cached package archive for %q: %v", importPath, err)
		}
		return nil // Cache miss.
	}
	defer f.Close()
	a, buildTime, err := compiler.ReadArchive(importPath, f, srcModTime, imports)
	if err != nil {
		log.Warningf("Failed to read cached package archive for %q: %v", importPath, err)
		return nil // Invalid/corrupted archive, cache miss.
	}
	if a == nil {
		log.Infof("Found out-of-date package archive for %q, built at %v.", importPath, buildTime)
		return nil // Archive is out-of-date, cache miss.
	}
	log.Infof("Found cached package archive for %q, built at %v.", importPath, buildTime)
	return a
}

// commonKey returns a part of the cache key common for all artifacts generated
// under a given BuildCache configuration.
func (bc *BuildCache) commonKey() string {
	return fmt.Sprintf("%#v + %v", *bc, compiler.Version)
}

// archiveKey returns a full cache key for a package's compiled archive.
func (bc *BuildCache) archiveKey(importPath string) string {
	return path.Join("archive", bc.commonKey(), importPath)
}
