package astutil

import (
	"go/token"
	"testing"

	"github.com/gopherjs/gopherjs/internal/srctesting"
)

func TestImportsUnsafe(t *testing.T) {
	tests := []struct {
		desc    string
		imports string
		want    bool
	}{
		{
			desc:    "no imports",
			imports: "",
			want:    false,
		}, {
			desc:    "other imports",
			imports: `import "some/other/package"`,
			want:    false,
		}, {
			desc:    "only unsafe",
			imports: `import "unsafe"`,
			want:    true,
		}, {
			desc: "multi-import decl",
			imports: `import (
				"some/other/package"
				"unsafe"
			)`,
			want: true,
		}, {
			desc: "two import decls",
			imports: `import "some/other/package"
			import "unsafe"`,
			want: true,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			src := "package testpackage\n\n" + test.imports
			fset := token.NewFileSet()
			file := srctesting.Parse(t, fset, src)
			got := ImportsUnsafe(file)
			if got != test.want {
				t.Fatalf("ImportsUnsafe() returned %t, want %t", got, test.want)
			}
		})
	}
}

func TestFuncKey(t *testing.T) {
	tests := []struct {
		desc string
		src  string
		want string
	}{
		{
			desc: "top-level function",
			src:  `package testpackage; func foo() {}`,
			want: "foo",
		}, {
			desc: "top-level exported function",
			src:  `package testpackage; func Foo() {}`,
			want: "Foo",
		}, {
			desc: "method",
			src:  `package testpackage; func (_ myType) bar() {}`,
			want: "myType.bar",
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			fdecl := srctesting.ParseFuncDecl(t, test.src)
			if got := FuncKey(fdecl); got != test.want {
				t.Errorf("Got %q, want %q", got, test.want)
			}
		})
	}
}

func TestEndsWithReturn(t *testing.T) {
	tests := []struct {
		desc string
		src  string
		want bool
	}{
		{
			desc: "empty function",
			src:  `func foo() {}`,
			want: false,
		}, {
			desc: "implicit return",
			src:  `func foo() { a() }`,
			want: false,
		}, {
			desc: "explicit return",
			src:  `func foo() { a(); return }`,
			want: true,
		}, {
			desc: "labelled return",
			src:  `func foo() { Label: return }`,
			want: true,
		}, {
			desc: "labelled call",
			src:  `func foo() { Label: a() }`,
			want: false,
		}, {
			desc: "return in a block",
			src:  `func foo() { a(); { b(); return; } }`,
			want: true,
		}, {
			desc: "a block without return",
			src:  `func foo() { a(); { b(); c(); } }`,
			want: false,
		}, {
			desc: "conditional block",
			src:  `func foo() { a(); if x { b(); return; } }`,
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			fdecl := srctesting.ParseFuncDecl(t, "package testpackage\n"+test.src)
			got := EndsWithReturn(fdecl.Body.List)
			if got != test.want {
				t.Errorf("EndsWithReturn() returned %t, want %t", got, test.want)
			}
		})
	}
}
