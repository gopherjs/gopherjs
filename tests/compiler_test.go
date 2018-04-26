package tests

import (
	"testing"
)

func TestVariadicNil(t *testing.T) {
	printVari := func(strs ...string) []string {
		return strs
	}

	if got := printVari(); got != nil {
		t.Errorf("expected printVari() to be %v; got: %v", nil, got)
	}
}
