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

			result := squeeze(input)
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

func TestPruneImports(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: `no imports`,
			src: `package testpackage
				func foo() {}`,
			want: `package testpackage
				func foo() {}`,
		}, {
			name: `keep used imports`,
			src: `package testpackage
				import "fmt"
				func foo() { fmt.Println("foo") }`,
			want: `package testpackage
				import "fmt"
				func foo() { fmt.Println("foo") }`,
		}, {
			name: `remove imports that are not used`,
			src: `package testpackage
				import "fmt"
				func foo() { }`,
			want: `package testpackage
				func foo() { }`,
		}, {
			name: `remove imports that are unused but masked by an object`,
			src: `package testpackage
				import "fmt"
				var fmt = "format"
				func foo() string { return fmt }`,
			want: `package testpackage
				var fmt = "format"
				func foo() string { return fmt }`,
		}, {
			name: `remove imports from empty file`,
			src: `package testpackage
				import "fmt"
				import _ "unsafe"`,
			want: `package testpackage`,
		}, {
			name: `remove imports from empty file except for unsafe when linking`,
			src: `package testpackage
				import "fmt"
				import "embed"

				//go:linkname foo runtime.foo
				import "unsafe"`,
			want: `package testpackage

				//go:linkname foo runtime.foo
				import _ "unsafe"`,
		}, {
			name: `keep embed imports when embedding`,
			src: `package testpackage
				import "fmt"
				import "embed"
				import "unsafe"

				//go:embed "foo.txt"
				var foo string`,
			want: `package testpackage
				import _ "embed"

				//go:embed "foo.txt"
				var foo string`,
		}, {
			name: `keep imports that just needed an underscore`,
			src: `package testpackage
				import "embed"
				//go:linkname foo runtime.foo
				import "unsafe"
				//go:embed "foo.txt"
				var foo string`,
			want: `package testpackage
				import _ "embed"
				//go:linkname foo runtime.foo
				import _ "unsafe"
				//go:embed "foo.txt"
				var foo string`,
		}, {
			name: `keep imports without names`,
			src: `package testpackage
				import _ "fmt"
				import "log"
				import . "math"

				var foo string`,
			want: `package testpackage
				import _ "fmt"

				import . "math"

				var foo string`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			st := srctesting.New(t)

			srcFile := st.Parse(`testSrc.go`, test.src)
			PruneImports(srcFile)
			got := srctesting.Format(t, st.FileSet, srcFile)

			// parse and format the expected result so that formatting matches
			wantFile := st.Parse(`testWant.go`, test.want)
			want := srctesting.Format(t, st.FileSet, wantFile)

			if got != want {
				t.Errorf("Unexpected resulting AST after PruneImports:\n\tgot:  %q\n\twant: %q", got, want)
			}
		})
	}
}

func TestFinalizeRemovals(t *testing.T) {
	tests := []struct {
		name       string
		src        string
		perforator func(f *ast.File)
		want       string
	}{
		{
			name: `no removals`,
			src: `package testpackage
				// foo took a journey
				func foo() {}
				// bar went home
				func bar[T any](v T) T { return v }
				// baz is a mystery
				var baz int = 42`,
			perforator: func(f *ast.File) {},
			want: `package testpackage
				// foo took a journey
				func foo() {}
				// bar went home
				func bar[T any](v T) T { return v }
				// baz is a mystery
				var baz int = 42`,
		}, {
			name: `removal first decl`,
			src: `package testpackage
				// foo took a journey
				func foo() {}
				// bar went home
				func bar[T any](v T) T { return v }
				// baz is a mystery
				var baz int = 42`,
			perforator: func(f *ast.File) {
				f.Decls[0] = nil
			},
			want: `package testpackage
				// bar went home
				func bar[T any](v T) T { return v }
				// baz is a mystery
				var baz int = 42`,
		}, {
			name: `removal middle decl`,
			src: `package testpackage
				// foo took a journey
				func foo() {}
				// bar went home
				func bar[T any](v T) T { return v }
				// baz is a mystery
				var baz int = 42`,
			perforator: func(f *ast.File) {
				f.Decls[1] = nil
			},
			want: `package testpackage
				// foo took a journey
				func foo() {}
				// baz is a mystery
				var baz int = 42`,
		}, {
			name: `removal last decl`,
			src: `package testpackage
				// foo took a journey
				func foo() {}
				// bar went home
				func bar[T any](v T) T { return v }
				// baz is a mystery
				var baz int = 42`,
			perforator: func(f *ast.File) {
				f.Decls[len(f.Decls)-1] = nil
			},
			want: `package testpackage
				// foo took a journey
				func foo() {}
				// bar went home
				func bar[T any](v T) T { return v }`,
		}, {
			name: `removal one whole value spec`,
			src: `package testpackage
				var (
					foo string   = "foo"
					bar, baz int = 42, 36
				)`,
			perforator: func(f *ast.File) {
				f.Decls[0].(*ast.GenDecl).Specs[1] = nil
			},
			want: `package testpackage
				var (
					foo string = "foo"
				)`,
		}, {
			name: `removal part of one value spec`,
			src: `package testpackage
				var (
					foo string   = "foo"
					bar, baz int = 42, 36
				)`,
			perforator: func(f *ast.File) {
				spec := f.Decls[0].(*ast.GenDecl).Specs[1].(*ast.ValueSpec)
				spec.Names[1] = nil
				spec.Values[1] = nil
			},
			want: `package testpackage
				var (
					foo string = "foo"
					bar int    = 42
				)`,
		}, {
			name: `removal all parts of one value spec`,
			src: `package testpackage
				var (
					foo string   = "foo"
					bar, baz int = 42, 36
				)`,
			perforator: func(f *ast.File) {
				spec := f.Decls[0].(*ast.GenDecl).Specs[1].(*ast.ValueSpec)
				spec.Names[0] = nil
				spec.Values[0] = nil
				spec.Names[1] = nil
				spec.Values[1] = nil
			},
			want: `package testpackage
				var (
					foo string = "foo"
				)`,
		},
		{
			name: `removal all value specs`,
			src: `package testpackage
				var (
					foo string   = "foo"
					bar, baz int = 42, 36
				)`,
			perforator: func(f *ast.File) {
				decl := f.Decls[0].(*ast.GenDecl)
				decl.Specs[0] = nil
				decl.Specs[1] = nil
			},
			want: `package testpackage`,
		}, {
			name: `removal one type spec`,
			src: `package testpackage
				type (
					foo interface{ String() string }
					bar struct{ baz int }
				)`,
			perforator: func(f *ast.File) {
				decl := f.Decls[0].(*ast.GenDecl)
				decl.Specs[0] = nil
			},
			want: `package testpackage
				type (
					bar struct{ baz int }
				)`,
		}, {
			name: `removal all type specs`,
			src: `package testpackage
				type (
					foo interface{ String() string }
					bar struct{ baz int }
				)`,
			perforator: func(f *ast.File) {
				decl := f.Decls[0].(*ast.GenDecl)
				decl.Specs[0] = nil
				decl.Specs[1] = nil
			},
			want: `package testpackage`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			st := srctesting.New(t)

			srcFile := st.Parse(`testSrc.go`, test.src)
			test.perforator(srcFile)
			FinalizeRemovals(srcFile)
			got := srctesting.Format(t, st.FileSet, srcFile)

			// parse and format the expected result so that formatting matches
			wantFile := st.Parse(`testWant.go`, test.want)
			want := srctesting.Format(t, st.FileSet, wantFile)

			if got != want {
				t.Errorf("Unexpected resulting AST:\n\tgot:  %q\n\twant: %q", got, want)
			}
		})
	}
}

func TestConcatenateFiles(t *testing.T) {
	tests := []struct {
		name    string
		srcHead string
		srcTail string
		want    string
		expErr  string
	}{
		{
			name: `add a method with a comment`,
			srcHead: `package testpackage
					// foo is an original method.
					func foo() {}`,
			srcTail: `package testpackage
					// bar is a concatenated method
					// from an additional override file.
					func bar() {}`,
			want: `package testpackage
					// foo is an original method.
					func foo() {}
					// bar is a concatenated method
					// from an additional override file.
					func bar() {}`,
		}, {
			name: `merge existing singular unnamed imports`,
			srcHead: `package testpackage
					import "fmt"
					import "bytes"
					
					func prime(str fmt.Stringer) *bytes.Buffer {
						return bytes.NewBufferString(str.String())
					}`,
			srcTail: `package testpackage
					import "bytes"
					import "fmt"
					
					func cat(strs ...fmt.Stringer) fmt.Stringer {
						buf := &bytes.Buffer{}
						for _, str := range strs {
							buf.WriteString(str.String())
						}
						return buf
					}`,
			want: `package testpackage
					import (
						"bytes"
						"fmt"
					)
					
					func prime(str fmt.Stringer) *bytes.Buffer {
						return bytes.NewBufferString(str.String())
					}
					func cat(strs ...fmt.Stringer) fmt.Stringer {
						buf := &bytes.Buffer{}
						for _, str := range strs {
							buf.WriteString(str.String())
						}
						return buf
					}`,
		}, {
			name: `merge existing named imports`,
			srcHead: `package testpackage
					import (
						foo "fmt"
						bar "bytes"
					)
					func prime(str foo.Stringer) *bar.Buffer {
						return bar.NewBufferString(str.String())
					}`,
			srcTail: `package testpackage
					import (
						bar "bytes"
						foo "fmt"
					)
					func cat(strs ...foo.Stringer) foo.Stringer {
						buf := &bar.Buffer{}
						for _, str := range strs {
							buf.WriteString(str.String())
						}
						return buf
					}`,
			want: `package testpackage
					import (
						bar "bytes"
						foo "fmt"
					)
					
					func prime(str foo.Stringer) *bar.Buffer {
						return bar.NewBufferString(str.String())
					}
					func cat(strs ...foo.Stringer) foo.Stringer {
						buf := &bar.Buffer{}
						for _, str := range strs {
							buf.WriteString(str.String())
						}
						return buf
					}`,
		}, {
			name: `merge imports that don't overlap`,
			srcHead: `package testpackage
					import (
						"fmt"
						"bytes"
					)
					func prime(str fmt.Stringer) *bytes.Buffer {
						return bytes.NewBufferString(str.String())
					}`,
			srcTail: `package testpackage
					import "math"
					import "log"
					func NaNaNaBatman(name string, value float64) {
						if math.IsNaN(value) {
							log.Println("Warning: "+name+" is NaN")
						}
					}`,
			want: `package testpackage
					import (
						"bytes"
						"fmt"
						"log"
						"math"
					)
					func prime(str fmt.Stringer) *bytes.Buffer {
						return bytes.NewBufferString(str.String())
					}
					func NaNaNaBatman(name string, value float64) {
						if math.IsNaN(value) {
							log.Println("Warning: " + name + " is NaN")
						}
					}`,
		}, {
			name: `merge two package comments`,
			srcHead: `// Original package comment
					package testpackage
					func foo() {}`,
			srcTail: `// Additional package comment
					package testpackage
					var bar int`,
			want: `// Original package comment

					// Additional package comment
					package testpackage
					func foo() {}
					var bar int`,
		}, {
			name: `take package comment from tail`,
			srcHead: `package testpackage
					func foo() {}`,
			srcTail: `// Additional package comment
					package testpackage
					var bar int`,
			want: `// Additional package comment
					package testpackage
					func foo() {}
					var bar int`,
		}, {
			name: `packages with different package names`,
			srcHead: `package testpackage
					func foo() {}`,
			srcTail: `package otherTestPackage
					func bar() {}`,
			expErr: `can not concatenate files with different package names: "testpackage" != "otherTestPackage"`,
		}, {
			name: `import mismatch with one named`,
			srcHead: `package testpackage
					import "fmt"
					func foo() { fmt.Println("foo") }`,
			srcTail: `package testpackage
					import f1 "fmt"
					func bar() { f1.Println("bar") }`,
			expErr: `import from of "fmt" can not be concatenated with different name: "fmt" != "f1"`,
		}, {
			name: `import mismatch with both named`,
			srcHead: `package testpackage
					import f1 "fmt"
					func foo() { f1.Println("foo") }`,
			srcTail: `package testpackage
					import f2 "fmt"
					func bar() { f2.Println("bar") }`,
			expErr: `import from of "fmt" can not be concatenated with different name: "f1" != "f2"`,
		}, {
			name: `import mismatch with old being blank`,
			srcHead: `package testpackage
					import _ "unsafe"
					//go:linkname foo runtime.foo
					func bar()`,
			srcTail: `package testpackage
					import "unsafe"
					func foo() unsafe.Pointer { return nil }`,
			want: `package testpackage
					import "unsafe"
					//go:linkname foo runtime.foo
					func bar()
					func foo() unsafe.Pointer { return nil }`,
		}, {
			name: `import mismatch with new being blank`,
			srcHead: `package testpackage
					import "unsafe"
					func foo() unsafe.Pointer { return nil }`,
			srcTail: `package testpackage
					import _ "unsafe"
					//go:linkname foo runtime.foo
					func bar()`,
			want: `package testpackage
					import "unsafe"
					func foo() unsafe.Pointer { return nil }
					//go:linkname foo runtime.foo
					func bar()`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			st := srctesting.New(t)
			if (len(test.want) > 0) == (len(test.expErr) > 0) {
				t.Fatal(`One and only one of "want" and "expErr" must be set`)
			}

			headFile := st.Parse(`testHead.go`, test.srcHead)
			tailFile := st.Parse(`testTail.go`, test.srcTail)
			err := ConcatenateFiles(headFile, tailFile)
			if err != nil {
				if len(test.expErr) == 0 {
					t.Errorf(`Expected an AST but got an error: %v`, err)
				} else if err.Error() != test.expErr {
					t.Errorf("Unexpected error:\n\tgot:  %q\n\twant: %q", err.Error(), test.expErr)
				}
				return
			}

			// The formatter expects the comment line numbers to be consecutive
			// so that layout is preserved. We can't guarantee that the line
			// numbers are correct after appending the files, which is fine
			// as long as we aren't trying to format it.
			// Setting the file comments to nil will force the formatter to use
			// the comments on the AST nodes when the node is reached which
			// gives a more accurate view of the concatenated file.
			headFile.Comments = nil
			got := srctesting.Format(t, st.FileSet, headFile)
			if len(test.want) == 0 {
				t.Errorf("Expected an error but got AST:\n\tgot:  %q\n\twant: %q", got, test.expErr)
				return
			}

			// parse and format the expected result so that formatting matches.
			wantFile := st.Parse("testWant.go", test.want)
			want := srctesting.Format(t, st.FileSet, wantFile)
			if got != want {
				t.Errorf("Unexpected resulting AST:\n\tgot:  %q\n\twant: %q", got, want)
			}
		})
	}
}
