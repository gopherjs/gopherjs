//go:build js
// +build js

package bufio_test

import "testing"

func TestReadStringAllocs(t *testing.T) {
	t.Skip("Memory allocation counters are not available in GopherJS.")
}
