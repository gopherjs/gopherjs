package tests

import (
	"testing"

	_ "reflect"
	_ "unsafe"

	"github.com/google/go-cmp/cmp"
	"github.com/gopherjs/gopherjs/tests/testdata/linkname/method"
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

func TestLinknameMethods(t *testing.T) {
	defer func() {
		if err := recover(); err != nil {
			t.Fatalf("method.TestLinkname() paniced: %s", err)
		}
	}()
	method.TestLinkname(t)
}

type name struct{ bytes *byte }
type nameOff int32
type rtype struct{}

//go:linkname rtype_nameOff reflect.(*rtype).nameOff
func rtype_nameOff(r *rtype, off nameOff) name

//go:linkname newName reflect.newName
func newName(n, tag string, exported bool) name

//go:linkname name_name reflect.name.name
func name_name(name) string

//go:linkname resolveReflectName reflect.resolveReflectName
func resolveReflectName(n name) nameOff

func TestLinknameReflectName(t *testing.T) {
	info := "myinfo"
	off := resolveReflectName(newName(info, "", false))
	n := rtype_nameOff(nil, off)
	if s := name_name(n); s != info {
		t.Fatalf("to reflect.name got %q: want %q", s, info)
	}
}
