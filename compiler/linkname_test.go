package compiler

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func parseSource(t *testing.T, src string) (*ast.File, *token.FileSet) {
	t.Helper()

	const filename = "<src>"
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		t.Log(src)
		t.Fatalf("Failed to parse source code: %s", err)
	}
	return file, fset
}

func makePackage(t *testing.T, src string) *types.Package {
	t.Helper()

	file, fset := parseSource(t, src)
	conf := types.Config{Importer: importer.Default()}
	pkg, err := conf.Check(file.Name.Name, fset, []*ast.File{file}, nil)
	if err != nil {
		t.Log(src)
		t.Fatalf("Failed to type check source code: %s", err)
	}

	return pkg
}

func TestSymName(t *testing.T) {
	pkg := makePackage(t,
		`package testcase

	func AFunction() {}
	type AType struct {}
	func (AType) AMethod() {}
	func (AType) APointerMethod() {}
	var AVariable int32
	`)

	tests := []struct {
		obj  types.Object
		want SymName
	}{
		{
			obj:  pkg.Scope().Lookup("AFunction"),
			want: SymName{PkgPath: "testcase", Name: "AFunction"},
		}, {
			obj:  pkg.Scope().Lookup("AType"),
			want: SymName{PkgPath: "testcase", Name: "AType"},
		}, {
			obj:  types.NewMethodSet(pkg.Scope().Lookup("AType").Type()).Lookup(pkg, "AMethod").Obj(),
			want: SymName{PkgPath: "testcase", Name: "AType.AMethod"},
		}, {
			obj:  types.NewMethodSet(pkg.Scope().Lookup("AType").Type()).Lookup(pkg, "APointerMethod").Obj(),
			want: SymName{PkgPath: "testcase", Name: "AType.APointerMethod"},
		}, {
			obj:  pkg.Scope().Lookup("AVariable"),
			want: SymName{PkgPath: "testcase", Name: "AVariable"},
		},
	}

	for _, test := range tests {
		t.Run(test.obj.Name(), func(t *testing.T) {
			got := newSymName(test.obj)
			if got != test.want {
				t.Errorf("NewSymName(%q) returned %#v, want: %#v", test.obj.Name(), got, test.want)
			}
		})
	}
}

func TestParseGoLinknames(t *testing.T) {
	tests := []struct {
		desc           string
		src            string
		wantError      string
		wantDirectives []GoLinkname
	}{
		{
			desc: "no directives",
			src: `package testcase
			
			// This comment doesn't start with go:linkname
			func a() {}
			// go:linkname directive must have no space between the slash and the directive.
			func b() {}
			// An example in the middle of a comment is also not a directive: //go:linkname foo bar.baz
			func c() {}
			`,
			wantDirectives: []GoLinkname{},
		}, {
			desc: "normal use case",
			src: `package testcase

			import _ "unsafe"
			
			//go:linkname a other/package.testcase_a
			func a()
			`,
			wantDirectives: []GoLinkname{
				{
					Reference:      SymName{PkgPath: "testcase", Name: "a"},
					Implementation: SymName{PkgPath: "other/package", Name: "testcase_a"},
				},
			},
		}, {
			desc: "multiple directives in one comment group",
			src: `package testcase
			import _ "unsafe"

			// The following functions are implemented elsewhere:
			//go:linkname a other/package.a
			//go:linkname b other/package.b

			func a()
			func b()
			`,
			wantDirectives: []GoLinkname{
				{
					Reference:      SymName{PkgPath: "testcase", Name: "a"},
					Implementation: SymName{PkgPath: "other/package", Name: "a"},
				}, {
					Reference:      SymName{PkgPath: "testcase", Name: "b"},
					Implementation: SymName{PkgPath: "other/package", Name: "b"},
				},
			},
		}, {
			desc: "unsafe not imported",
			src: `package testcase
			
			//go:linkname a other/package.a
			func a()
			`,
			wantError: `import "unsafe"`,
		}, {
			desc: "gopherjs: both parameters are required",
			src: `package testcase
			
			import _ "unsafe"

			//go:linkname a
			func a()
			`,
			wantError: "usage",
		}, {
			desc: "referenced function doesn't exist",
			src: `package testcase
			
			import _ "unsafe"

			//go:linkname b other/package.b
			func a()
			`,
			wantError: `"b" is not found`,
		}, {
			desc: "gopherjs: referenced a variable, not a function",
			src: `package testcase
			
			import _ "unsafe"

			//go:linkname a other/package.a
			var a string = "foo"
			`,
			wantError: `is only supported for functions`,
		}, {
			desc: "gopherjs: can not insert local implementation",
			src: `package testcase
			
			import _ "unsafe"

			//go:linkname a other/package.a
			func a() { println("do a") }
			`,
			wantError: `can not insert local implementation`,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			file, fset := parseSource(t, test.src)
			directives, err := parseGoLinknames(fset, "testcase", file)

			if test.wantError != "" {
				if err == nil {
					t.Fatalf("ParseGoLinknames() returned no error, want: %s.", test.wantError)
				} else if !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("ParseGoLinknames() returned error: %s. Want an error containing %q.", err, test.wantError)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseGoLinkanmes() returned error: %s. Want: no error.", err)
			}

			if diff := cmp.Diff(test.wantDirectives, directives, cmpopts.EquateEmpty()); diff != "" {
				t.Fatalf("ParseGoLinknames() returned diff (-want,+got):\n%s", diff)
			}
		})
	}
}
