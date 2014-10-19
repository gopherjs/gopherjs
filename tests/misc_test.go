package tests

import (
	"testing"
)

func TestSyntax1(t *testing.T) {
	a := 42
	if *&*&a != 42 {
		t.Fail()
	}
}
