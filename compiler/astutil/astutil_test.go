package astutil

import (
	"go/ast"
	"strconv"
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
			file := srctesting.New(t).Parse("test.go", src)
			got := ImportsUnsafe(file)
			if got != test.want {
				t.Fatalf("ImportsUnsafe() returned %t, want %t", got, test.want)
			}
		})
	}
}

func TestImportName(t *testing.T) {
	tests := []struct {
		desc string
		src  string
		want string
	}{
		{
			desc: `named import`,
			src:  `import foo "some/other/bar"`,
			want: `foo`,
		}, {
			desc: `unnamed import`,
			src:  `import "some/other/bar"`,
			want: `bar`,
		}, {
			desc: `dot import`,
			src:  `import . "some/other/bar"`,
			want: ``,
		}, {
			desc: `blank import`,
			src:  `import _ "some/other/bar"`,
			want: ``,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			src := "package testpackage\n\n" + test.src
			file := srctesting.New(t).Parse("test.go", src)
			if len(file.Imports) != 1 {
				t.Fatal(`expected one and only one import`)
			}
			importSpec := file.Imports[0]
			got := ImportName(importSpec)
			if got != test.want {
				t.Fatalf(`ImportName() returned %q, want %q`, got, test.want)
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
			desc: `directive on specification, not on declaration`,
			src: `package testpackage;
				type (
					Foo int

					//gopherjs:do-stuff
					Bar struct{}
				)`,
			want: false,
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
			desc: `directive on declaration, not on specification`,
			src: `package testpackage;
				//gopherjs:do-stuff
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
			file := srctesting.New(t).Parse("test.go", test.src)
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

func TestSqueezeIdents(t *testing.T) {
	tests := []struct {
		desc   string
		count  int
		assign []int
	}{
		{
			desc:   `no squeezing`,
			count:  5,
			assign: []int{0, 1, 2, 3, 4},
		}, {
			desc:   `missing front`,
			count:  5,
			assign: []int{3, 4},
		}, {
			desc:   `missing back`,
			count:  5,
			assign: []int{0, 1, 2},
		}, {
			desc:   `missing several`,
			count:  10,
			assign: []int{1, 2, 3, 6, 8},
		}, {
			desc:   `empty`,
			count:  0,
			assign: []int{},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			input := make([]*ast.Ident, test.count)
			for _, i := range test.assign {
				input[i] = ast.NewIdent(strconv.Itoa(i))
			}

			result := Squeeze(input)
			if len(result) != len(test.assign) {
				t.Errorf("Squeeze() returned a slice %d long, want %d", len(result), len(test.assign))
			}
			for i, id := range input {
				if i < len(result) {
					if id == nil {
						t.Errorf(`Squeeze() returned a nil in result at %d`, i)
					} else {
						value, err := strconv.Atoi(id.Name)
						if err != nil || value != test.assign[i] {
							t.Errorf(`Squeeze() returned %s at %d instead of %d`, id.Name, i, test.assign[i])
						}
					}
				} else if id != nil {
					t.Errorf(`Squeeze() didn't clear out tail of slice, want %d nil`, i)
				}
			}
		})
	}
}
