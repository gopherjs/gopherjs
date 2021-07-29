//go:build js
// +build js

package edwards25519

import "testing"

func TestScalarMultDistributesOverAdd(t *testing.T) {
	t.Skip("slow") // Times out, takes ~13 minutes
}

func TestScalarMultNonIdentityPoint(t *testing.T) {
	t.Skip("slow") // Takes > 5 min
}

func TestScalarMultMatchesBaseMult(t *testing.T) {
	t.Skip("slow") // Takes > 5 min
}

func TestVarTimeDoubleBaseMultMatchesBaseMult(t *testing.T) {
	t.Skip("slow") // Times out in CI
}
