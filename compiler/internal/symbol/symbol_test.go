package symbol

import (
	"go/token"
	"go/types"
	"testing"

	"github.com/gopherjs/gopherjs/internal/srctesting"
)

func TestName(t *testing.T) {
	const src = `package testcase

	func AFunction() {}
	type AType struct {}
	func (AType) AMethod() {}
	func (*AType) APointerMethod() {}
	var AVariable int32
	`

	fset := token.NewFileSet()
	_, pkg := srctesting.Check(t, fset, srctesting.Parse(t, fset, src))

	tests := []struct {
		obj  types.Object
		want Name
	}{
		{
			obj:  pkg.Scope().Lookup("AFunction"),
			want: Name{PkgPath: "test", Name: "AFunction"},
		}, {
			obj:  pkg.Scope().Lookup("AType"),
			want: Name{PkgPath: "test", Name: "AType"},
		}, {
			obj:  types.NewMethodSet(pkg.Scope().Lookup("AType").Type()).Lookup(pkg, "AMethod").Obj(),
			want: Name{PkgPath: "test", Name: "AType.AMethod"},
		}, {
			obj:  types.NewMethodSet(types.NewPointer(pkg.Scope().Lookup("AType").Type())).Lookup(pkg, "APointerMethod").Obj(),
			want: Name{PkgPath: "test", Name: "(*AType).APointerMethod"},
		}, {
			obj:  pkg.Scope().Lookup("AVariable"),
			want: Name{PkgPath: "test", Name: "AVariable"},
		},
	}

	for _, test := range tests {
		t.Run(test.obj.Name(), func(t *testing.T) {
			got := New(test.obj)
			if got != test.want {
				t.Errorf("NewSymName(%q) returned %#v, want: %#v", test.obj.Name(), got, test.want)
			}
		})
	}
}
