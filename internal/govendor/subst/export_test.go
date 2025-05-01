package subst

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"
)

func TestNestedSubst(t *testing.T) {
	const source = `
		package P

		func A[T any](){
			type B struct{X T}
			type C[U any] struct{X T; Y U}
		}`

	fSet := token.NewFileSet()
	f, err := parser.ParseFile(fSet, "hello.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}

	var conf types.Config
	pkg, err := conf.Check("P", fSet, []*ast.File{f}, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		fnName string   // the name of the nesting function
		fnArgs []string // type expressions of args for the nesting function
		stName string   // the name of the named type
		stArgs []string // type expressions of args for the named type
		want   string   // expected underlying value after substitution
	}{
		{
			fnName: "A", fnArgs: []string{"int"},
			stName: "B", stArgs: []string{},
			want: "struct{X int}",
		},
	} {
		ctxt := types.NewContext()

		fnGen, _ := pkg.Scope().Lookup(test.fnName).(*types.Func)
		if fnGen == nil {
			t.Fatal("Failed to find the function " + test.fnName)
		}
		fnType := fnGen.Type().(*types.Signature)
		fnArgs := evalTypeList(t, fSet, pkg, test.fnArgs)
		fnInst, err := types.Instantiate(ctxt, fnType, fnArgs, true)
		if err != nil {
			t.Fatalf("Failed to instantiate %s: %v", fnType, err)
		}
		fnFunc := types.NewFunc(fnGen.Pos(), pkg, fnGen.Name(), fnInst.(*types.Signature))

		stType, _ := fnFunc.Scope().Lookup(test.stName).Type().(*types.Named)
		if stType == nil {
			t.Fatal("Failed to find the object " + test.fnName + " in function " + test.fnName)
		}
		stArgs := evalTypeList(t, fSet, pkg, test.stArgs)

		stSubst := New(types.NewContext(), fnFunc, stType.TypeParams(), stArgs)
		stInst := stSubst.Type(stType.Underlying())

		if got := stInst.String(); got != test.want {
			t.Errorf("subst{%v->%v}.typ(%s) = %v, want %v", test.stName, test.stArgs, stType.Underlying(), got, test.want)
		}
	}
}

func evalType(t *testing.T, fSet *token.FileSet, pkg *types.Package, expr string) types.Type {
	tv, err := types.Eval(fSet, pkg, 0, expr)
	if err != nil {
		t.Fatalf("Eval(%s) failed: %v", expr, err)
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
