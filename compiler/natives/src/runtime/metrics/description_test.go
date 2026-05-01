//go:build js

package metrics_test

import "testing"

// GopherJS does not record any runtime metrics. The original test calls
// `runtime_readMetricNames“ (a `//go:linkname` forward declaration that
// resolves to `runtime.readMetricNames` in Go), which is not implemented
// in GopherJS's runtime and would panic with "native function not implemented".
// `runtime_readMetricNames` is only called by TestNames so doesn't need to
// be overridden.
//
//gopherjs:replace
func TestNames(t *testing.T) {
	t.Skip("runtime metrics are not supported by GopherJS")
}
