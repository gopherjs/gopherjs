//go:build js

package strings_test

import "testing"

func TestBuilderAllocs(t *testing.T) {
	t.Skip("runtime.ReadMemStats, testing.AllocsPerRun not supported in GopherJS")
}

func TestBuilderGrow(t *testing.T) {
	t.Skip("runtime.ReadMemStats, testing.AllocsPerRun not supported in GopherJS")
}

func TestCompareStrings(t *testing.T) {
	t.Skip("unsafeString not supported in GopherJS")
}

func TestClone(t *testing.T) {
	t.Skip("conversion to reflect.StringHeader is not supported in GopherJS")
}
