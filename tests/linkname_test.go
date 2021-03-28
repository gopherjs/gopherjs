package tests

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/gopherjs/gopherjs/tests/testdata/linkname/one"
)

func TestLinknames(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("one.DoAll() paniced: %s", err)
		}
	}()
	want := "doing one\n" +
		"doing two\n" +
		"doing imported one: doing internal one: one secret\n" +
		"doing three\n" +
		"doing imported three: doing internal three: three secret\n"
	got := one.DoAll()

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("Callink linknamed functions returned a diff (-want,+got):\n%s", diff)
	}
}
