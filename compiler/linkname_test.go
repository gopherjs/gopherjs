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
	"github.com/gopherjs/gopherjs/compiler/internal/symbol"
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

func TestParseGoLinknames(t *testing.T) {
	tests := []struct {
		desc           string
		pkgPath        string
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
					Reference:      symbol.Name{PkgPath: "testcase", Name: "a"},
					Implementation: symbol.Name{PkgPath: "other/package", Name: "testcase_a"},
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
					Reference:      symbol.Name{PkgPath: "testcase", Name: "a"},
					Implementation: symbol.Name{PkgPath: "other/package", Name: "a"},
				}, {
					Reference:      symbol.Name{PkgPath: "testcase", Name: "b"},
					Implementation: symbol.Name{PkgPath: "other/package", Name: "b"},
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
			desc: "gopherjs: ignore one-argument linknames",
			src: `package testcase
			
			import _ "unsafe"

			//go:linkname a
			func a()
			`,
			wantDirectives: []GoLinkname{},
		}, {
			desc: `gopherjs: linkname has too many arguments`,
			src: `package testcase
			
			import _ "unsafe"

			//go:linkname a other/package.a too/many.args
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
			desc:    `gopherjs: ignore know referenced variables`,
			pkgPath: `reflect`,
			src: `package reflect
			
			import _ "unsafe"

			//go:linkname zeroVal other/package.zeroVal
			var zeroVal []bytes
			`,
			wantDirectives: []GoLinkname{},
		}, {
			desc: "gopherjs: can not insert local implementation",
			src: `package testcase
			
			import _ "unsafe"

			//go:linkname a other/package.a
			func a() { println("do a") }
			`,
			wantError: `can not insert local implementation`,
		}, {
			desc:    `gopherjs: ignore known local implementation insert`,
			pkgPath: `runtime`, // runtime is known and ignored
			src: `package runtime
			
			import _ "unsafe"

			//go:linkname a other/package.a
			func a() { println("do a") }
			`,
			wantDirectives: []GoLinkname{},
		}, {
			desc: `gopherjs: link to function with receiver`,
			// //go:linkname <localname> <importpath>.<type>.<name>
			src: `package testcase
			
			import _ "unsafe"

			//go:linkname a other/package.b.a
			func a()
			`,
			wantDirectives: []GoLinkname{
				{
					Reference:      symbol.Name{PkgPath: `testcase`, Name: `a`},
					Implementation: symbol.Name{PkgPath: `other/package`, Name: `b.a`},
				},
			},
		}, {
			desc: `gopherjs: link to function with pointer receiver`,
			// //go:linkname <localname> <importpath>.<(*type)>.<name>
			src: `package testcase
			
			import _ "unsafe"

			//go:linkname a other/package.*b.a
			func a()
			`,
			wantDirectives: []GoLinkname{
				{
					Reference:      symbol.Name{PkgPath: `testcase`, Name: `a`},
					Implementation: symbol.Name{PkgPath: `other/package`, Name: `*b.a`},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			file, fset := parseSource(t, test.src)
			pkgPath := `testcase`
			if len(test.pkgPath) > 0 {
				pkgPath = test.pkgPath
			}
			directives, err := parseGoLinknames(fset, pkgPath, file)

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
