//go:build js
// +build js

package rand_test

import "testing"

func TestFloat32(t *testing.T) {
	t.Skip("slow")
}

func TestConcurrent(t *testing.T) {
	t.Skip("using nosync")
}
