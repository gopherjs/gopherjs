//go:build js
// +build js

package embedtest

import (
	"testing"
)

func TestUninitialized(t *testing.T) {
	t.Skip("reflect.DeepEqual error for empty slice and variadic")
}
