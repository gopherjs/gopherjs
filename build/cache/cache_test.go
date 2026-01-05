package cache

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type CacheableMock struct{ Data string }

func (m *CacheableMock) Write(encode func(any) error) error { return encode(m.Data) }
func (m *CacheableMock) Read(decode func(any) error) error  { return decode(&m.Data) }

func TestStore(t *testing.T) {
	cacheForTest(t)

	const data = `fake/data`
	const importPath = `fake/package`
	want := &CacheableMock{Data: data}
	srcModTime := newTime(0.0)
	buildTime := newTime(5.0)
	bc := BuildCache{}
	if bc.Load(want, importPath, srcModTime) {
		t.Errorf("Got: %s was found in the cache with %q. Want: empty cache.", importPath, want.Data)
	}

	if !bc.Store(want, importPath, buildTime) {
		t.Errorf("Failed to store %s with %q.", importPath, want.Data)
	}

	got := &CacheableMock{}
	if !bc.Load(got, importPath, srcModTime) {
		t.Errorf("Got: %s was not found in the cache. Want: package found.", importPath)
	} else {
		if diff := cmp.Diff(want, got); len(diff) > 0 {
			t.Errorf("Loaded package is different from stored (-want,+got):\n%s", diff)
		}
	}

	// Make sure the package names are a part of the cache key.
	got = &CacheableMock{}
	if bc.Load(got, "fake/other", srcModTime) {
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
		}, {
			cache1: BuildCache{Version: "1.19.0-beta2+go1.19.13"},
			cache2: BuildCache{Version: "1.18.0+go1.18.10"},
		},
	}

	srcModTime := newTime(0.0)
	buildTime := newTime(5.0)
	for _, test := range tests {
		const data = `fake/data`
		const importPath = `fake/package`
		s0 := &CacheableMock{Data: data}
		if !test.cache1.Store(s0, importPath, buildTime) {
			t.Errorf("Failed to store cache for cache1: %#v", test.cache1)
			continue
		}

		s1 := &CacheableMock{}
		if test.cache2.Load(s1, importPath, srcModTime) {
			t.Logf("-cache1,+cache2:\n%s", cmp.Diff(test.cache1, test.cache2))
			t.Errorf("Got: %v loaded from cache. Want: build parameter change invalidates cache.", s1)
		}
	}
}

func TestOldCache(t *testing.T) {
	cacheForTest(t)

	const data = `fake/data`
	const importPath = "fake/package"
	want := &CacheableMock{Data: data}
	buildTime := newTime(5.0)
	bc := BuildCache{}
	if !bc.Store(want, importPath, buildTime) {
		t.Errorf("Failed to store %s with %q.", importPath, want.Data)
	}

	oldSrcModTime := newTime(2.0) // older than cache build time, so cache is up-to-date
	got := &CacheableMock{}
	if !bc.Load(got, importPath, oldSrcModTime) || got.Data != want.Data {
		t.Errorf("Got: cache with %q. Want: up-to-date package cache to be loaded with %q.", got.Data, want.Data)
	}

	newerSrcModTime := newTime(7.0) // newer than cache build time, so cache is stale
	got = &CacheableMock{}
	if bc.Load(got, importPath, newerSrcModTime) || len(got.Data) != 0 {
		t.Errorf("Got: cache was not nil with %q. Want: stale package cache with %q to not be loaded with.", got.Data, want.Data)
	}
}

func TestSkipOfTestPackage(t *testing.T) {
	cacheForTest(t)

	const data = `fake/data`
	const importPath = "fake/package"
	want := &CacheableMock{Data: data}
	srcModTime := newTime(0.0)
	buildTime := newTime(5.0)

	bc := BuildCache{}
	if !bc.Store(want, importPath, buildTime) {
		t.Errorf("Failed to store %s with %q.", importPath, want.Data)
	}

	// Simulate writing a cache for a pacakge under test.
	bc.TestedPackage = importPath
	if bc.Store(want, importPath, buildTime) {
		t.Errorf("Got: cache stored for %q. Want: test packages to not write to cache.", importPath)
	}
	if bc.Store(want, importPath+"_test", buildTime) {
		t.Errorf("Got: cache stored for %q. Want: test packages to not write to cache.", importPath+"_test")
	}

	// Simulate reading the cache for a pacakge under test.
	got := &CacheableMock{}
	if bc.Load(got, importPath, srcModTime) {
		t.Errorf("Got: cache with %q. Want: test package cache to not be loaded for %q.", got.Data, importPath)
	}
	got = &CacheableMock{}
	if bc.Load(got, importPath+"_test", srcModTime) {
		t.Errorf("Got: cache with %q. Want: test package cache to not be loaded for %q.", got.Data, importPath+"_test")
	}

	// No package under test, cache should work normally and load previously stored non-test package.
	bc.TestedPackage = ""
	got = &CacheableMock{}
	if !bc.Load(got, importPath, srcModTime) || got.Data != want.Data {
		t.Errorf("Got: cache with %q. Want: up-to-date package cache to be loaded with %q.", got.Data, want.Data)
	}
}

func cacheForTest(t *testing.T) {
	t.Helper()
	originalRoot := cacheRoot
	t.Cleanup(func() { cacheRoot = originalRoot })
	cacheRoot = t.TempDir()
}

func newTime(seconds float64) time.Time {
	return time.Date(1969, time.July, 20, 20, 17, 0, 0, time.UTC).
		Add(time.Duration(seconds * float64(time.Second)))
}
