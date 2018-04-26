package tests

import (
	"testing"
)

func TestVariadicNil(t *testing.T) {
	printVari := func(strs ...string) []string {
		return strs
	}

	if got := printVari(); got != nil {
		t.Errorf("printVari(): got: %#v; want %#v.", got, nil)
	}
}
