# Compiler directives

Compiler directives allow to provide low-level instructions to the GopherJS
compiler, which are outside of the Go language itself. Compiler directives are
specific to each Go compiler implementation and may be a source of portability
issues, so it is recommended to avoid using them if possible.

GopherJS compiler supports the following directives:

- [go:linkname](#golinkname)
- [go:embed](#goembed)
- [gopherjs:keep-original](#gopherjskeep-original)
- [gopherjs:purge](#gopherjspurge)
- [gopherjs:override-signature](#gopherjsoverride-signature)

## `go:linkname`

This is a limited version of the `go:linkname` directive the upstream Go
compiler implements. Usage:

```go
import _ "unsafe" // for go:linkname

//go:linkname localname import/path.remotename
func localname(arg1 type1, arg2 type2) (returnType, error)
```

This directive has an effect of making a `remotename` function from
`import/path` package available to the current package as `localname`.
Signatures of `remotename` and `localname` must be identical. Since this
directive can subvert package incapsulation, the source file that uses the
directive must also import `unsafe`.

The following directive formats are supported:

- `//go:linkname <localname> <importpath>.<name>`
- `//go:linkname <localname> <importpath>.<type>.<name>`
- `//go:linkname <localname> <importpath>.<(*type)>.<name>`

Compared to the upstream Go, the following limitations exist in GopherJS:

- The directive only works on package-level functions or methods (variables
  are not supported).
- The directive can only be used to "import" implementation from another
  package, and not to "provide" local implementation to another package.

See [gopherjs/issues/1000](https://github.com/gopherjs/gopherjs/issues/1000)
for details.

## `go:embed`

This is a very similar version of the `go:embed` directive the upstream Go
compiler implements.
GopherJS leverages [goembed](https://github.com/visualfc/goembed)
to parse this directive and provide support reading embedded content. Usage:

```go
import _ "embed" // for go:embed

//go:embed externalText
var embeddedText string

//go:embed externalContent
var embeddedContent []byte

//go:embed file1
//go:embed file2
// ...
//go:embed image/* blobs/*
var embeddedFiles embed.FS
```

This directive affects the variable specification (e.g. `embeddedText`)
that the comment containing the directive is associated with.
There may be one embed directives associated with `string` or `[]byte`
variables. There may be one or more embed directives associated with
`embed.FS` variables and each directive may contain one or more
file matching patterns. The effect is that the variable will be assigned to
the content (e.g. `externalText`) given in the directive. In the case
of `embed.FS`, several embedded files will be accessible.

See [pkg.go.dev/embed](https://pkg.go.dev/embed#hdr-Directives)
for more information.

## `gopherjs:keep-original`

This directive is custom to GopherJS. This directive can be added to a
function declaration in the native file overrides as part of the build step.

This will keep the original function by the same name as the function
in the overrides, however it will prepend `_gopherjs_original_` to the original
function's name. This allows the original function to be called by functions
in the overrides and the overridden function to be called instead of the
original. This is useful when wanting to augment the original behavior without
having to rewrite the entire original function. Usage:

```go
//gopherjs:keep-original
func foo(a, b int) int {
  return _gopherjs_original_foo(a+1, b+1) - 1
}
```

## `gopherjs:purge`

This directive is custom to GopherJS. This directive can be added
to most declarations and specification in the native file overrides as
part of the build step.
This can be added to structures, interfaces, methods, functions,
variables, or constants, but are not supported for imports, structure fields,
nor interface function signatures.

This will remove the original structure, interface, etc from both the override
files and the original files.
If this is added to a structure, then all functions in the original files
that use that structure as a receiver will also be removed.
This is useful for removing all the code that is invalid in GopherJS,
such as code using unsupported features (e.g. generic interfaces before
generics were fully supported). In many cases the overrides to replace
the original code may not have use of all the original functions and
variables or the original code is not intended to be replaced yet.
Usage:

```go
//gopherjs:purge
var data string

//gopherjs:purge
// This will also purge any function starting with `dataType` as the receiver.
type dataType struct {}

//gopherjs:purge
type interfaceType any

//gopherjs:purge
func doThing[T ~string](value T)
```

## `gopherjs:override-signature`

This directive is custom to GopherJS. This directive can be added to a
function declaration in the native file overrides as part of the build step.

This will remove the function from the overrides but record the signature
used in the overrides, then update the original function with that signature
provided in the overrides.
The affect is to change the receiver, type parameters,
parameters, or return types of the original function. The original function
and override function must have the same function key name so that they can
be associated, meaning the identifier of the receiver, if there is one, must
match and the identifier of the function must match.

This allows the signature to be modified without modifying the body of a
function thus allowing the types to be adjusted to work in GopherJS.
The signature may need to be replaced because it uses a parameter type
that is invalid in GopherJS or the signature uses unsupported features
(e.g. generic interfaces before generics were fully supported).
Usage:

```go
// -- in original file --
func Foo[T comparable](a, b T) (T, bool) {
  if a == b {
    return a, true
  }
  return b, false
}

// -- in override file --
//gopherjs:override-signature
func Foo(a, b any) (any, bool)

// -- result in augmented original --
func Foo(a, b any) (any, bool) {
  if a == b {
    return a, true
  }
  return b, false
}
```

```go
// -- in original file --
func (f *Foo[A, B, C]) Bar(a int, b *A) (*A, error) {
  //...
}

// -- in override file --
//gopherjs:override-signature
func (f *Foo) Bar(a int, b jsTypeA) (jsTypeA, error)

// -- result in augmented original --
func (f *Foo) Bar(a int, b jsTypeA) (jsTypeA, error) {
  //...
}
```
