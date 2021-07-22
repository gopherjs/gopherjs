//go:build js
// +build js

package reflectlite_test

import (
	. "internal/reflectlite"
	"testing"
)

func TestTypes(t *testing.T) {
	for i, tt := range typeTests {
		if i == 30 {
			continue
		}
		testReflectType(t, i, Field(ValueOf(tt.i), 0).Type(), tt.s)
	}
}

func TestNameBytesAreAligned(t *testing.T) {
	t.Skip("TestNameBytesAreAligned")
}
