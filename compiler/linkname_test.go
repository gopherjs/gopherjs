package compiler

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func makePackage(t *testing.T, src string) *types.Package {
	t.Helper()
	const filename = "<src>"
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		t.Log(src)
		t.Fatalf("Failed to parse source code: %s", err)
	}
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
			got := NewSymName(test.obj)
			if got != test.want {
				t.Errorf("NewSymName(%q) returned %#v, want: %#v", test.obj.Name(), got, test.want)
			}
		})
	}
}
