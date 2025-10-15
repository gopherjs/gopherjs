//go:build js

package rand_test

import "testing"

func TestConcurrent(t *testing.T) {
	t.Skip("using nosync")
}
