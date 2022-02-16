package cache

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gopherjs/gopherjs/compiler"
)

func TestStore(t *testing.T) {
	cacheForTest(t)

	want := &compiler.Archive{
		ImportPath: "fake/package",
		Imports:    []string{"fake/dep"},
	}

	bc := BuildCache{}
	if got := bc.LoadArchive(want.ImportPath); got != nil {
		t.Errorf("Got: %s was found in the cache. Want: empty cache.", got.ImportPath)
	}
	bc.StoreArchive(want)
	got := bc.LoadArchive(want.ImportPath)
	if got == nil {
		t.Errorf("Got: %s wan not found in the cache. Want: archive is can be loaded after store.", want.ImportPath)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Loaded archive is different from stored (-want,+got):\n%s", diff)
	}

	// Make sure the package names are a part of the cache key.
	if got := bc.LoadArchive("fake/other"); got != nil {
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

	for _, test := range tests {
		a := &compiler.Archive{ImportPath: "package/fake"}
		test.cache1.StoreArchive(a)

		if got := test.cache2.LoadArchive(a.ImportPath); got != nil {
			t.Logf("-cache1,+cache2:\n%s", cmp.Diff(test.cache1, test.cache2))
			t.Errorf("Got: %v loaded from cache. Want: build parameter change invalidates cache.", got)
		}
	}
}

func cacheForTest(t *testing.T) {
	t.Helper()
	originalRoot := cacheRoot
	t.Cleanup(func() { cacheRoot = originalRoot })
	cacheRoot = t.TempDir()
}
