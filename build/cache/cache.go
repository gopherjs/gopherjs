// Package cache solves one of the hardest computer science problems in
// application to GopherJS compiler outputs.
package cache

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"go/build"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
)

// Cacheable defines methods to serialize and deserialize cachable objects.
// This object should represent a package's build artifact.
//
// The encode and decode functions are typically wrappers around gob.Encoder.Encode
// and gob.Decoder.Decode, but other formats are possible as well, the same way
// as FileSet.Write and FileSet.Read work with any encode/decode functions.
type Cacheable interface {
	Write(encode func(any) error) error
	Read(decode func(any) error) error
}

// Cache defines methods to store and load cacheable objects.
type Cache interface {

	// Store stores the package with the given import path in the cache.
	// Any error inside this method will cause the cache not to be persisted.
	//
	// The passed in buildTime is used to determine if the package is out-of-date when reloaded.
	// Typically it should be set to the srcModTime or time.Now().
	Store(c Cacheable, importPath string, buildTime time.Time) bool

	// Load reads a previously cached package at the given import path,
	// if it was previously stored.
	//
	// The loaded package would have been built with the same configuration as
	// the build cache was.
	Load(c Cacheable, importPath string, srcModTime time.Time) bool
}

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

var _ Cache = (*BuildCache)(nil)

// BuildCache manages build artifacts that are cached for incremental builds.
//
// Cache is designed to be non-durable: any store and load errors are swallowed
// and simply lead to a cache miss. The caller must be able to handle cache
// misses. Nil pointer to BuildCache is valid and simply disables caching.
//
// BuildCache struct fields represent build parameters which change invalidates
// the cache. For example, any artifacts that were cached for a Linux build
// must not be reused for a non-Linux build. GopherJS version change also
// invalidates the cache. It is callers responsibility to ensure that artifacts
// passed the Store function were generated with the same build
// parameters as the cache is configured.
//
// There is no upper limit for the total cache size. It can be cleared
// programmatically via the Clear() function, or the user can just delete the
// directory if it grows too big.
//
// The cached files are gzip compressed, therefore each file uses the gzip
// checksum as a basic integrity check performed after reading the file.
//
// TODO(nevkontakte): changes in the input sources or dependencies doesn't
// currently invalidate the cache. This is handled at the higher level by
// checking cached package timestamp against loaded package modification time.
type BuildCache struct {
	GOOS      string
	GOARCH    string
	GOROOT    string
	GOPATH    string
	BuildTags []string

	// Version should be set to compiler.Version
	Version string

	// TestedPackage is the import path of the package being tested, or
	// empty when not building for tests. The package under test is built
	// with *_test.go sources included so we should always skip reading
	// and writing cache in that case. Since we are caching prior to
	// type-checking for generics, any package importing the package under
	// test should be unaffected.
	TestedPackage string
}

func (bc BuildCache) String() string {
	return fmt.Sprintf("%#v", bc)
}

func (bc *BuildCache) isTestPackage(importPath string) bool {
	return bc != nil && len(importPath) > 0 &&
		(importPath == bc.TestedPackage || importPath == bc.TestedPackage+"_test")
}

func (bc *BuildCache) Store(c Cacheable, importPath string, buildTime time.Time) bool {
	if bc == nil {
		return false // Caching is disabled.
	}
	if bc.isTestPackage(importPath) {
		log.Infof("Skipped storing cache of test package for %q.", importPath)
		return false // Don't use cache when building the package under test.
	}

	start := time.Now()
	path := cachedPath(bc.packageKey(importPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		log.Warningf("Failed to create build cache directory: %v", err)
		return false
	}
	// Write the package in a temporary file first to avoid concurrency errors.
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path))
	if err != nil {
		log.Warningf("Failed to temporary build cache file: %v", err)
		return false
	}
	defer f.Close()
	if err := bc.serialize(c, buildTime, f); err != nil {
		log.Warningf("Failed to write build cache package %q: %v", importPath, err)
		// Make sure we don't leave a half-written package behind.
		os.Remove(f.Name())
		return false
	}
	f.Close()
	// Rename fully written file into its permanent name.
	if err := os.Rename(f.Name(), path); err != nil {
		log.Warningf("Failed to rename build cache package %q to %q: %v", importPath, path, err)
		return false
	}
	dur := time.Since(start).Round(time.Millisecond)
	log.Infof("Successfully stored build package %q as %q (%v).", importPath, path, dur)
	return true
}

func (bc *BuildCache) Load(c Cacheable, importPath string, srcModTime time.Time) bool {
	if bc == nil {
		return false // Caching is disabled.
	}
	if bc.isTestPackage(importPath) {
		log.Infof("Skipped loading cache of test package for %q.", importPath)
		return false // Don't use cache when building the package under test.
	}

	start := time.Now()
	path := cachedPath(bc.packageKey(importPath))
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Infof("No cached package for %q at %q.", importPath, path)
		} else {
			log.Warningf("Failed to open cached package for %q at %q: %v", importPath, path, err)
		}
		return false // Cache miss.
	}
	defer f.Close()
	buildTime, old, err := bc.deserialize(c, srcModTime, f)
	if err != nil {
		log.Warningf("Failed to read cached package for %q at %q: %v", importPath, path, err)
		return false // Invalid/corrupted package, cache miss.
	}
	if old {
		log.Infof("Found out-of-date package for %q, built at %v.", importPath, buildTime)
		return false // Cache miss, package is out-of-date.
	}
	dur := time.Since(start).Round(time.Millisecond)
	log.Infof("Found cached package for %q, built at %v (%v).", importPath, buildTime, dur)
	return true
}

func (bc *BuildCache) serialize(c Cacheable, buildTime time.Time, w io.Writer) (err error) {
	zw := gzip.NewWriter(w)
	defer func() {
		// This close flushes the gzip but does not close the given writer.
		if closeErr := zw.Close(); err == nil {
			err = closeErr
		}
	}()

	ge := gob.NewEncoder(zw)
	if err := ge.Encode(buildTime); err != nil {
		return err
	}
	return c.Write(ge.Encode)
}

func (bc *BuildCache) deserialize(c Cacheable, srcModTime time.Time, r io.Reader) (buildTime time.Time, old bool, err error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return buildTime, false, err
	}
	defer func() {
		// This close checks the gzip checksum but does not close the given reader.
		if closeErr := zr.Close(); err == nil {
			err = closeErr
		}
	}()

	gd := gob.NewDecoder(zr)
	if err := gd.Decode(&buildTime); err != nil {
		return buildTime, false, err
	}
	if srcModTime.After(buildTime) {
		return buildTime, true, nil // Package is out-of-date, cache miss.
	}
	return buildTime, false, c.Read(gd.Decode)
}

// commonKey returns a part of the cache key common for all artifacts generated
// under a given BuildCache configuration.
func (bc *BuildCache) commonKey() string {
	type commonKey struct {
		GOOS      string
		GOARCH    string
		GOROOT    string
		GOPATH    string
		BuildTags []string
		Version   string
	}
	// These are the values that affect the files that are included into a
	// package's source via build constraints.
	ck := commonKey{
		GOOS:      bc.GOOS,
		GOARCH:    bc.GOARCH,
		GOROOT:    bc.GOROOT,
		GOPATH:    bc.GOPATH,
		BuildTags: bc.BuildTags,
		Version:   bc.Version,
	}
	return fmt.Sprintf("%#v", ck)
}

// packageKey returns a full cache key for a package's cache.
func (bc *BuildCache) packageKey(importPath string) string {
	return path.Join("package", bc.commonKey(), importPath)
}
