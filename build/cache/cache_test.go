package cache

import (
	"go/types"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/gopherjs/gopherjs/compiler"
)

func TestStore(t *testing.T) {
	cacheForTest(t)

	want := &compiler.Archive{
		ImportPath: "fake/package",
		Imports:    []string{"fake/dep"},
	}

	srcModTime := newTime(0.0)
	buildTime := newTime(5.0)
	imports := map[string]*types.Package{}
	bc := BuildCache{}
	if got := bc.LoadArchive(want.ImportPath, srcModTime, imports); got != nil {
		t.Errorf("Got: %s was found in the cache. Want: empty cache.", got.ImportPath)
	}
	bc.StoreArchive(want, buildTime)
	got := bc.LoadArchive(want.ImportPath, srcModTime, imports)
	if got == nil {
		t.Errorf("Got: %s was not found in the cache. Want: archive is can be loaded after store.", want.ImportPath)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Loaded archive is different from stored (-want,+got):\n%s", diff)
	}

	// Make sure the package names are a part of the cache key.
	if got := bc.LoadArchive("fake/other", srcModTime, imports); got != nil {
		t.Errorf("Got: fake/other was found in cache: %#v. Want: nil for packages that weren't cached.", got)
	}
}

func TestInvalidation(t *testing.T) {
	cacheForTest(t)

	tests := []struct {
		cache1 BuildCache
		cache2 BuildCache
	}{
		{
			cache1: BuildCache{Minify: true},
			cache2: BuildCache{Minify: false},
		}, {
			cache1: BuildCache{GOOS: "dos"},
			cache2: BuildCache{GOOS: "amiga"},
		}, {
			cache1: BuildCache{GOARCH: "m68k"},
			cache2: BuildCache{GOARCH: "mos6502"},
		}, {
			cache1: BuildCache{GOROOT: "here"},
			cache2: BuildCache{GOROOT: "there"},
		}, {
			cache1: BuildCache{GOPATH: "home"},
			cache2: BuildCache{GOPATH: "away"},
		},
	}

	srcModTime := newTime(0.0)
	buildTime := newTime(5.0)
	imports := map[string]*types.Package{}
	for _, test := range tests {
		a := &compiler.Archive{ImportPath: "package/fake"}
		test.cache1.StoreArchive(a, buildTime)

		if got := test.cache2.LoadArchive(a.ImportPath, srcModTime, imports); got != nil {
			t.Logf("-cache1,+cache2:\n%s", cmp.Diff(test.cache1, test.cache2))
			t.Errorf("Got: %v loaded from cache. Want: build parameter change invalidates cache.", got)
		}
	}
}

func TestOldArchive(t *testing.T) {
	cacheForTest(t)

	want := &compiler.Archive{
		ImportPath: "fake/package",
		Imports:    []string{"fake/dep"},
	}

	buildTime := newTime(5.0)
	imports := map[string]*types.Package{}
	bc := BuildCache{}
	bc.StoreArchive(want, buildTime)

	oldSrcModTime := newTime(2.0) // older than archive build time, so archive is up-to-date
	got := bc.LoadArchive(want.ImportPath, oldSrcModTime, imports)
	if got == nil {
		t.Errorf("Got: %s was nil. Want: up-to-date archive to be loaded.", want.ImportPath)
	}

	newerSrcModTime := newTime(7.0) // newer than archive build time, so archive is stale
	got = bc.LoadArchive(want.ImportPath, newerSrcModTime, imports)
	if got != nil {
		t.Errorf("Got: %s was not nil. Want: stale archive to not be loaded.", want.ImportPath)
	}
}

func cacheForTest(t *testing.T) {
	t.Helper()
	originalRoot := cacheRoot
	t.Cleanup(func() { cacheRoot = originalRoot })
	cacheRoot = t.TempDir()
}

func newTime(seconds float64) time.Time {
	return time.Date(1969, 7, 20, 20, 17, 0, 0, time.UTC).
		Add(time.Duration(seconds * float64(time.Second)))
}
