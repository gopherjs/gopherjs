package tags

import "testing"

func TestPlusBuild(t *testing.T) {
	// Simple tests.
	assert(t, true, `// +build linux`, `linux`, `arm`)
	assert(t, true, `// +build !windows`, `linux`, `arm`)
	assert(t, false, `// +build !windows`, `windows`, `amd64`)

	// Build tags separated by a space are OR-ed together.
	assert(t, true, `// +build arm 386`, `arm`, `amd64`)
	assert(t, true, `// +build arm 386`, `linux`, `386`)
	assert(t, false, `// +build arm 386`, `linux`, `amd64`)

	// Build tags separated by a comma are AND-ed together.
	assert(t, true, `// +build !windows,!plan9`, `linux`, `386`)
	assert(t, false, `// +build !windows,!plan9`, `windows`, `amd64`)
	assert(t, false, `// +build !windows,!plan9`, `plan9`, `386`)

	// Build tags on multiple lines are AND-ed together.
	assert(t, true, "// +build !windows\n// +build amd64", `linux`, `amd64`)
	assert(t, false, "// +build !windows\n// +build amd64", `windows`, `amd64`)
	assert(t, false, "// +build !windows\n// +build amd64", `linux`, `386`)

	// Test that (!a OR !b) matches anything.
	assert(t, true, `// +build !windows !plan9`, `windows`, `amd64`)

	// GOPHERJS: Custom rule, test that don't run on nacl should also not run on js.
	assert(t, false, `// +build !nacl,!plan9,!windows`, `darwin`, `js`)
}

func TestGoBuild(t *testing.T) {
	// Simple tests.
	assert(t, true, `//go:build linux`, `linux`, `arm`)
	assert(t, true, `//go:build !windows`, `linux`, `arm`)
	assert(t, false, `//go:build !windows`, `windows`, `amd64`)

	// Build tags OR-ed together.
	assert(t, false, `//go:build arm || 386`, `linux`, `amd64`)

	// Build tags AND-ed together.
	assert(t, true, `//go:build !windows && !plan9`, `linux`, `386`)
	assert(t, false, `//go:build !windows && !plan9`, `windows`, `amd64`)
	assert(t, false, `//go:build !windows && !plan9`, `plan9`, `386`)

	// Build tags on multiple lines are AND-ed together.
	assert(t, true, "//go:build !windows\n//go:build amd64", `linux`, `amd64`)
	assert(t, false, "//go:build !windows\n//go:build amd64", `windows`, `amd64`)
	assert(t, false, "//go:build !windows\n//go:build amd64", `linux`, `386`)

	// Test that (!a OR !b) matches anything.
	assert(t, true, `//go:build !windows || !plan9`, `windows`, `amd64`)

	// GOPHERJS: Custom rule, test that don't run on nacl should also not run on js.
	assert(t, false, `//go:build !nacl && !plan9 && !windows`, `darwin`, `js`)
}

func TestOther(t *testing.T) {
	// A file with no build tags will always be tested.
	assert(t, true, `// This is a test.`, `os`, `arch`)

	// A file with malformed build tags will be skipped.
	assert(t, false, `// +build linux?`, `linux`)
	assert(t, false, `//go:build linux?`, `linux`)

	// Build constraint must appear before the package clause so ignore any after it.
	assert(t, true, "//go:build 386\npackage tags\n//go:build !linux", `linux`, `386`)
	assert(t, false, "//go:build 386\npackage tags\n//go:build !linux", `linux`, `amd64`)
}

func assert(t *testing.T, expMatch bool, src string, tags ...string) {
	isMatch, line := Match(src, tags...)
	if isMatch != expMatch {
		t.Logf(`tags: %v`, tags)
		t.Logf(`constraint: %q`, line)
		t.Errorf(`expected Match to return %t`, expMatch)
	}
}
