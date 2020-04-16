package build

import (
	"testing"
)

func TestAugmenter(t *testing.T) {
	s, err := NewSession(Options{})
	if err != nil {
		t.Fatalf("failed to create build session: %s", err)
	}
	pkgs, err := s.load("runtime")
	if err != nil {
		t.Fatalf("failed to load package 'runtime': %s", err)
	}
	pkg := pkgs[0]

	a := NewAugmenter()
	err = a.Augment(pkg)
	if err != nil {
		t.Errorf("failed to augment package %s: %s", pkg, err)
	}
	t.Logf("Files parsed: %d", len(pkg.Syntax))
}
