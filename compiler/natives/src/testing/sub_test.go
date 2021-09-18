//go:build js
// +build js

package testing

func TestBenchmarkReadMemStatsBeforeFirstRun(t *T) {
	t.Skip("runtime.ReadMemStats() is not supported by GopherJS.")
}

func TestTRun(t *T) {
	// TODO(nevkontakte): This test performs string comparisons expecting to find
	// sub_test.go in the output, but GopherJS currently reports caller
	// locations as "test.<random_number>" due to minimal caller and source map support.
	t.Skip("GopherJS doesn't support source maps sufficiently.")
}

func TestBRun(t *T) {
	// TODO(nevkontakte): This test performs string comparisons expecting to find
	// sub_test.go in the output, but GopherJS currently reports caller
	// locations as "test.<random_number>" due to minimal caller and source map support.
	t.Skip("GopherJS doesn't support source maps sufficiently.")
}
