package astutil

import (
	"fmt"
	"go/ast"
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
			desc: `top-level function`,
			src:  `func foo() {}`,
			want: `foo`,
		}, {
			desc: `top-level exported function`,
			src:  `func Foo() {}`,
			want: `Foo`,
		}, {
			desc: `method on reference`,
			src:  `func (_ myType) bar() {}`,
			want: `myType.bar`,
		}, {
			desc: `method on pointer`,
			src:  ` func (_ *myType) bar() {}`,
			want: `myType.bar`,
		}, {
			desc: `method on generic reference`,
			src:  ` func (_ myType[T]) bar() {}`,
			want: `myType.bar`,
		}, {
			desc: `method on generic pointer`,
			src:  ` func (_ *myType[T]) bar() {}`,
			want: `myType.bar`,
		}, {
			desc: `method on struct with multiple generics`,
			src:  ` func (_ *myType[T1, T2, T3, T4]) bar() {}`,
			want: `myType.bar`,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			src := `package testpackage; ` + test.src
			fdecl := srctesting.ParseFuncDecl(t, src)
			if got := FuncKey(fdecl); got != test.want {
				t.Errorf(`Got %q, want %q`, got, test.want)
			}
		})
	}
}

func TestPruneOriginal(t *testing.T) {
	tests := []struct {
		desc string
		src  string
		want bool
	}{
		{
			desc: "no comment",
			src: `package testpackage;
			func foo() {}`,
			want: false,
		}, {
			desc: "regular godoc",
			src: `package testpackage;
			// foo does something
			func foo() {}`,
			want: false,
		}, {
			desc: "only directive",
			src: `package testpackage;
			//gopherjs:prune-original
			func foo() {}`,
			want: true,
		}, {
			desc: "directive with explanation",
			src: `package testpackage;
			//gopherjs:prune-original because reasons
			func foo() {}`,
			want: true,
		}, {
			desc: "directive in godoc",
			src: `package testpackage;
			// foo does something
			//gopherjs:prune-original
			func foo() {}`,
			want: true,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			fdecl := srctesting.ParseFuncDecl(t, test.src)
			if got := PruneOriginal(fdecl); got != test.want {
				t.Errorf("PruneOriginal() returned %t, want %t", got, test.want)
			}
		})
	}
}

func TestHasDirectiveOnDecl(t *testing.T) {
	tests := []struct {
		desc string
		src  string
		want bool
	}{
		{
			desc: `no comment on function`,
			src: `package testpackage;
				func foo() {}`,
			want: false,
		}, {
			desc: `no directive on function with comment`,
			src: `package testpackage;
				// foo has no directive
				func foo() {}`,
			want: false,
		}, {
			desc: `wrong directive on function`,
			src: `package testpackage;
				//gopherjs:wrong-directive
				func foo() {}`,
			want: false,
		}, {
			desc: `correct directive on function`,
			src: `package testpackage;
				//gopherjs:do-stuff
				// foo has a directive to do stuff
				func foo() {}`,
			want: true,
		}, {
			desc: `correct directive in multiline comment on function`,
			src: `package testpackage;
				/*gopherjs:do-stuff
				  foo has a directive to do stuff
				*/
				func foo() {}`,
			want: true,
		}, {
			desc: `invalid directive in multiline comment on function`,
			src: `package testpackage;
				/*
				gopherjs:do-stuff
				*/
				func foo() {}`,
			want: false,
		}, {
			desc: `prefix directive on function`,
			src: `package testpackage;
				//gopherjs:do-stuffs
				func foo() {}`,
			want: false,
		}, {
			desc: `multiple directives on function`,
			src: `package testpackage;
				//gopherjs:wrong-directive
				//gopherjs:do-stuff
				//gopherjs:another-directive
				func foo() {}`,
			want: true,
		}, {
			desc: `directive with explanation on function`,
			src: `package testpackage;
				//gopherjs:do-stuff 'cause we can
				func foo() {}`,
			want: true,
		}, {
			desc: `no directive on type declaration`,
			src: `package testpackage;
				// Foo has a comment
				type Foo int`,
			want: false,
		}, {
			desc: `directive on type declaration`,
			src: `package testpackage;
				//gopherjs:do-stuff
				type Foo int`,
			want: true,
		}, {
			desc: `no directive on const declaration`,
			src: `package testpackage;
				const foo = 42`,
			want: false,
		}, {
			desc: `directive on const documentation`,
			src: `package testpackage;
				//gopherjs:do-stuff
				const foo = 42`,
			want: true,
		}, {
			desc: `no directive on var declaration`,
			src: `package testpackage;
				var foo = 42`,
			want: false,
		}, {
			desc: `directive on var documentation`,
			src: `package testpackage;
				//gopherjs:do-stuff
				var foo = 42`,
			want: true,
		}, {
			desc: `no directive on var declaration`,
			src: `package testpackage;
				import _ "embed"`,
			want: false,
		}, {
			desc: `directive on var documentation`,
			src: `package testpackage;
				//gopherjs:do-stuff
				import _ "embed"`,
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			const action = `do-stuff`
			decl := srctesting.ParseDecl(t, test.src)
			if got := hasDirective(decl, action); got != test.want {
				t.Errorf(`hasDirective(%T, %q) returned %t, want %t`, decl, action, got, test.want)
			}
		})
	}
}

func TestHasDirectiveOnSpec(t *testing.T) {
	tests := []struct {
		desc string
		src  string
		want bool
	}{
		{
			desc: `no directive on type specification`,
			src: `package testpackage;
				type Foo int`,
			want: false,
		}, {
			desc: `directive in doc on type specification`,
			src: `package testpackage;
				type (
					//gopherjs:do-stuff
					Foo int
				)`,
			want: true,
		}, {
			desc: `directive in line on type specification`,
			src: `package testpackage;
				type Foo int //gopherjs:do-stuff`,
			want: true,
		}, {
			desc: `no directive on const specification`,
			src: `package testpackage;
				const foo = 42`,
			want: false,
		}, {
			desc: `directive in doc on const specification`,
			src: `package testpackage;
				const (
					//gopherjs:do-stuff
					foo = 42
				)`,
			want: true,
		}, {
			desc: `directive in line on const specification`,
			src: `package testpackage;
				const foo = 42 //gopherjs:do-stuff`,
			want: true,
		}, {
			desc: `no directive on var specification`,
			src: `package testpackage;
				var foo = 42`,
			want: false,
		}, {
			desc: `directive in doc on var specification`,
			src: `package testpackage;
				var (
					//gopherjs:do-stuff
					foo = 42
				)`,
			want: true,
		}, {
			desc: `directive in line on var specification`,
			src: `package testpackage;
				var foo = 42 //gopherjs:do-stuff`,
			want: true,
		}, {
			desc: `no directive on import specification`,
			src: `package testpackage;
				import _ "embed"`,
			want: false,
		}, {
			desc: `directive in doc on import specification`,
			src: `package testpackage;
				import (
					//gopherjs:do-stuff
					_ "embed"
				)`,
			want: true,
		}, {
			desc: `directive in line on import specification`,
			src: `package testpackage;
				import _ "embed" //gopherjs:do-stuff`,
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			const action = `do-stuff`
			spec := srctesting.ParseSpec(t, test.src)
			if got := hasDirective(spec, action); got != test.want {
				t.Errorf(`hasDirective(%T, %q) returned %t, want %t`, spec, action, got, test.want)
			}
		})
	}
}

func TestHasDirectiveOnFile(t *testing.T) {
	tests := []struct {
		desc string
		src  string
		want bool
	}{
		{
			desc: `no directive on file`,
			src: `package testpackage;
				//gopherjs:do-stuff
				type Foo int`,
			want: false,
		}, {
			desc: `directive on file`,
			src: `//gopherjs:do-stuff
				package testpackage;
				type Foo int`,
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			const action = `do-stuff`
			fset := token.NewFileSet()
			file := srctesting.Parse(t, fset, test.src)
			if got := hasDirective(file, action); got != test.want {
				t.Errorf(`hasDirective(%T, %q) returned %t, want %t`, file, action, got, test.want)
			}
		})
	}
}

func TestHasDirectiveOnField(t *testing.T) {
	tests := []struct {
		desc string
		src  string
		want bool
	}{
		{
			desc: `no directive on struct field`,
			src: `package testpackage;
				type Foo struct {
					bar int
				}`,
			want: false,
		}, {
			desc: `directive in doc on struct field`,
			src: `package testpackage;
				type Foo struct {
					//gopherjs:do-stuff
					bar int
				}`,
			want: true,
		}, {
			desc: `directive in line on struct field`,
			src: `package testpackage;
				type Foo struct {
					bar int //gopherjs:do-stuff
				}`,
			want: true,
		}, {
			desc: `no directive on interface method`,
			src: `package testpackage;
				type Foo interface {
					Bar(a int) int
				}`,
			want: false,
		}, {
			desc: `directive in doc on interface method`,
			src: `package testpackage;
				type Foo interface {
					//gopherjs:do-stuff
					Bar(a int) int
				}`,
			want: true,
		}, {
			desc: `directive in line on interface method`,
			src: `package testpackage;
				type Foo interface {
					Bar(a int) int //gopherjs:do-stuff
				}`,
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			const action = `do-stuff`
			spec := srctesting.ParseSpec(t, test.src)
			tspec := spec.(*ast.TypeSpec)
			var field *ast.Field
			switch typeNode := tspec.Type.(type) {
			case *ast.StructType:
				field = typeNode.Fields.List[0]
			case *ast.InterfaceType:
				field = typeNode.Methods.List[0]
			default:
				t.Errorf(`unexpected node type, %T, when finding field`, typeNode)
				return
			}
			if got := hasDirective(field, action); got != test.want {
				t.Errorf(`hasDirective(%T, %q) returned %t, want %t`, field, action, got, test.want)
			}
		})
	}
}

func TestHasDirectiveBadCase(t *testing.T) {
	tests := []struct {
		desc string
		node any
		want string
	}{
		{
			desc: `untyped nil node`,
			node: nil,
			want: `unexpected node type to get doc from: <nil>`,
		}, {
			desc: `unexpected node type`,
			node: &ast.ArrayType{},
			want: `unexpected node type to get doc from: *ast.ArrayType`,
		}, {
			desc: `nil expected node type`,
			node: (*ast.FuncDecl)(nil),
			want: `<nil>`, // no panic
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			const action = `do-stuff`
			var got string
			func() {
				defer func() { got = fmt.Sprint(recover()) }()
				hasDirective(test.node, action)
			}()
			if got != test.want {
				t.Errorf(`hasDirective(%T, %q) returned %s, want %s`, test.node, action, got, test.want)
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
