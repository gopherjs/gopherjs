# Compiler directives

Compiler directives allow to provide low-level instructions to the GopherJS
compiler, which are outside of the Go language itself. Compiler directives are
specific to each Go compiler implementation and may be a source of portability
issues, so it is recommended to avoid using them if possible.

GopherJS compiler supports the following directives:

- [go:linkname](#golinkname)
- [go:embed](#goembed)
- [gopherjs:new](#gopherjsnew)
- [gopherjs:replace](#gopherjsreplace)
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

## `gopherjs:new`

This directive is custom to GopherJS. This directive can be added
to most declarations and specification in the native file overrides as
part of the build step. This does not work on imports.

This is an optional directive to help check for issues when upgrading to
newer versions of Go. This will indicate that the code it is on, is new code
being added to the package via the overrides.

When the overrides are being applied, if an identifier matches then the
code is replaced with the override, otherwise the code is added as new code.
If this directive is added to the override, it will raise an error when some
code by the same identifier exists. This allows the developer to indicate
they intended to add code without it being used as a replacement.

This is useful when upgrading to new Go versions. The changes to the Go code
could change or add an identifier that matches. Instead of the override
unexpectantly becoming a replacement, this will prevent that and let the
developers performing the upgrade know that they need to rename the override
to keep is as new code and not replace the Go code.

In the following example, suppose we added a function `clearData` for
go1.X. When upgrating to go1.Y, the developers of Go add thier own function
called `clearData`. Instead of quietly replacing the new Go code, this
directive would error to indicate we need to rename our new code to something
like `jsClearData` to prevent our code from replacing the Go code unintentinally.

```Go
//gopherjs:new
func clearData(obj *js.Object) { ... }

//gopherjs:new
var listOfData = []string { ... }
```

## `gopherjs:replace`

This directive is custom to GopherJS. This directive can be added
to most declarations and specification in the native file overrides as
part of the build step. This does not work on imports.

This is an optional directive to help check for issues when upgrading to
newer versions of Go. This will indicate that the code it is on, is replacing
code in a package.

When the overrides are being applied, if an identifier matches then the
code is replaced with the override, otherwise the code is added as new code.
If this directive is added to the override, it will raise an error when
some code by the same identifier does not exist. This allows the developer
to indicate they intende to replcae native code with this override and
prevent it from being added as new code.

This is useful when upgrading to new Go versions. The changes to the Go code
could change or remove an identifier that matches. Instead of the override
unexpectantly becoming new code, this will prevent that and let the developers
performing the upgrade know that they need to rename or remove the override
to keep the override replacing the correct Go code.

In the following example, suppose we were replacing a function `clearData` for
go1.X. When upgrading to go1.Y, the developers of Go renamed thier own
`clearData` function to `clearTableData`. Instead of quietly adding `clearData`
as a new function (which will likely just be removed with dead code elimination)
and not providing the intended override of the `clearTableData` function,
this directive would error to indicate we need to rename our override to match
the changes in Go.

```Go
//gopherjs:replace
func clearData(obj *js.Object) { ... }

//gopherjs:replace
var listOfData = []string { ... }
```

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

This will work like a `gopherjs:replace` as well, where if the native code
does not have this identifier, an error will occur. That helps a developer
know why the `_gopherjs_original_` code would be missing during an upgrade
to a new version of Go where the Go developers may have renamed or removed
the original code.

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

This will work like a `gopherjs:replace` as well, where if the native code
does not have this identifier, an error will occur. This helps developers
during an upgrade of Go versions that the Go developers may have renamed or
removed the original code being purged. If the code has been removed
we can remove unused override code. If the code was renamed, we can move the
purge to the new identifier.

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

This will work like a `gopherjs:replace` as well, where if the native code
does not have this identifier, an error will occur. This helps developers
during an upgrade of Go versions that the Go developers may have renamed or
removed the original code being overridden.
