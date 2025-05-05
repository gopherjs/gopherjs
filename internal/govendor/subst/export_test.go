package subst

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestNestedSubstInGenericFunction(t *testing.T) {
	const source = `
		package P

		func A[T any](){
			type B struct{X T}
			type C[U any] struct{X T; Y U}
		}
			
		func D(){
			type E[V any] struct{X V}
		}
		
		type F[W any] struct{X W}
		`

	fSet := token.NewFileSet()
	f, err := parser.ParseFile(fSet, `hello.go`, source, 0)
	if err != nil {
		t.Fatal(err)
	}

	var conf types.Config
	pkg, err := conf.Check("P", fSet, []*ast.File{f}, nil)
	if err != nil {
		t.Fatal(err)
	}

	type namedType struct {
		name string   // the name of the named type
		args []string // type expressions of args for the named type
	}

	for i, test := range []struct {
		nesting   []namedType
		want      string // expected underlying value after substitution
		substWant string // expected string value of the Subster
	}{
		// "Substituting types.Signatures with generic functions are currently unsupported."
		// since we should be getting back concrete functions from the type checker.
		//{
		//	nesting: []namedType{
		//		{name: `A`, args: []string{`int`}},
		//	},
		//},
		{
			nesting: []namedType{
				{name: `A`, args: []string{`int`}},
				{name: `B`},
			},
			want:      `struct{X int}`,
			substWant: `{T->int}`,
		},
		{
			nesting: []namedType{
				{name: `A`, args: []string{`int`}},
				{name: `C`, args: []string{`bool`}},
			},
			want:      `struct{X int; Y bool}`,
			substWant: `{T->int}:{U->bool}`,
		},
		{
			nesting: []namedType{
				{name: `D`},
			},
			want:      `func()`,
			substWant: `{}`,
		},
		{
			nesting: []namedType{
				{name: `D`},
				{name: `E`, args: []string{`int`}},
			},
			want:      `struct{X int}`,
			substWant: `{V->int}`,
		},
		{
			nesting: []namedType{
				{name: `F`, args: []string{`int`}},
			},
			want:      `struct{X int}`,
			substWant: `{W->int}`,
		},
	} {
		if len(test.nesting) == 0 {
			t.Fatalf(`Test %d: Must have at least one names type to instantiate`, i)
		}

		ctxt := types.NewContext()
		var subst *Subster
		var obj types.Object
		scope := pkg.Scope()
		for _, nt := range test.nesting {
			obj = scope.Lookup(nt.name)
			if obj == nil {
				t.Fatalf(`Test %d: Failed to find %s in package scope`, i, nt.name)
			}
			if fn, ok := obj.(*types.Func); ok {
				scope = fn.Scope()
			}
			args := evalTypeList(t, fSet, pkg, nt.args)
			tp := getTypeParams(t, obj.Type())
			subst = New(ctxt, tp, args, subst)
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf(`Test %d: panicked: %v`, i, r)
				}
			}()

			stInst := subst.Type(obj.Type().Underlying())
			if got := stInst.String(); got != test.want {
				t.Errorf("Test %d: %s.typ(%s): got %v, want %v", i, subst, obj.Type().Underlying(), got, test.want)
			}
			if got := subst.String(); got != test.substWant {
				t.Errorf("Test %d: subst string got %v, want %v", i, got, test.substWant)
			}
		}()
	}
}

func getTypeParams(t *testing.T, typ types.Type) *types.TypeParamList {
	switch typ := typ.(type) {
	case *types.Named:
		return typ.TypeParams()
	case *types.Signature:
		if tp := typ.RecvTypeParams(); tp != nil && tp.Len() > 0 {
			return tp
		}
		return typ.TypeParams()
	case interface{ Elem() types.Type }:
		// Pointer, slice, array, map, and channel types.
		return getTypeParams(t, typ.Elem())
	default:
		t.Fatalf(`getTypeParams(%v) hit unexpected type`, typ)
	}
	return nil
}

func evalType(t *testing.T, fSet *token.FileSet, pkg *types.Package, expr string) types.Type {
	tv, err := types.Eval(fSet, pkg, 0, expr)
	if err != nil {
		t.Fatalf(`Eval(%s) failed: %v`, expr, err)
	}
	return tv.Type
}

func evalTypeList(t *testing.T, fSet *token.FileSet, pkg *types.Package, exprs []string) []types.Type {
	typesList := make([]types.Type, 0, len(exprs))
	for _, expr := range exprs {
		typesList = append(typesList, evalType(t, fSet, pkg, expr))
	}
	return typesList
}
