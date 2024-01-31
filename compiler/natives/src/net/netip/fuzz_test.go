//go:build js
// +build js

package netip_test

import "testing"

func checkStringParseRoundTrip(t *testing.T, x interface{}, parse interface{}) {
	// TODO(nevkontakte): This function requires generics to function.
	// Re-enable after https://github.com/gopherjs/gopherjs/issues/1013 is resolved.
}
