# Compiler directives

Compiler directives allow to provide low-level instructions to the GopherJS
compiler, which are outside of the Go language itself. Compiler directives are
specific to each Go compiler implementation and may be a source of portability
issues, so it is recommended to avoid using them if possible.

GopherJS compiler supports the following directives:

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

Compared to the upstream Go, the following limitations exist in GopherJS:

  - The directive only works on package-level functions (variables and methods
    are not supported).
  - The directive can only be used to "import" implementation from another
    package, and not to "provide" local implementation to another package.

See https://github.com/gopherjs/gopherjs/issues/1000 for details.
