package build

import (
	"crypto/sha256"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
)

// cachePath is the base path for GopherJS's own build cache.
//
// It serves a similar function to the Go build cache, but is a lot more
// simplistic and therefore not compatible with Go. We use this cache directory
// to store build artifacts for packages loaded from a module, for which PkgObj
// provided by go/build points inside the module source tree, which can cause
// inconvenience with version control, etc.
var cachePath = func() string {
	path, err := os.UserCacheDir()
	if err == nil {
		return filepath.Join(path, "gopherjs", "build_cache")
	}

	return filepath.Join(build.Default.GOPATH, "pkg", "gopherjs_build_cache")
}()

// cachedPath returns a location inside the build cache for a given PkgObj path
// returned by go/build.
func cachedPath(orig string) string {
	if orig == "" {
		panic("CachedPath() must not be used with an empty string")
	}
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte(orig)))
	return filepath.Join(cachePath, sum[0:2], sum)
}
