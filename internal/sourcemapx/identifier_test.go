package sourcemapx

import (
	"go/token"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestIdentifier_String(t *testing.T) {
	ident := Identifier{
		Name:         "Foo$1",
		OriginalName: "Foo",
		OriginalPos:  token.Pos(42),
	}

	got := ident.String()
	if got != ident.Name {
		t.Errorf("Got: ident.String() = %q. Want: %q.", got, ident.Name)
	}
}

func TestIdentifier_EncodeHint(t *testing.T) {
	original := Identifier{
		Name:         "Foo$1",
		OriginalName: "Foo",
		OriginalPos:  token.Pos(42),
	}

	encoded := original.EncodeHint()
	hint, _ := ReadHint([]byte(encoded))
	decoded, err := hint.Unpack()
	if err != nil {
		t.Fatalf("Got: hint.Unpack() returned error: %s. Want: no error.", err)
	}
	if diff := cmp.Diff(original, decoded); diff != "" {
		t.Fatalf("Decoded hint differs from the original (-want,+got):\n%s", diff)
	}
}
