// +build !go1.13

package tests

import (
	"testing"
	"vendored"
)

func TestVendoring(t *testing.T) {
	if vendored.Answer != 42 {
		t.Fail()
	}
}
