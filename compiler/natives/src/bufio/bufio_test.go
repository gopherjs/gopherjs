//go:build js

package bufio_test

import "testing"

//gopherjs:replace
func TestReadStringAllocs(t *testing.T) {
	t.Skip("Memory allocation counters are not available in GopherJS.")
}
