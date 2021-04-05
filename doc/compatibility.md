# GopherJS compatibility

_TL;DR: GopherJS aims to provide full compatibility with regular Go, but JavaScript runtime introduces unavoidable differences._

Go ecosystem is broad and complex, which means there are several dimensions in which different levels of compatibility can be achieved:

 1. **[Go Language Specification](https://golang.org/ref/spec)**: full compatibility. With the exception of several minor differences documented below, GopherJS _should_ be fully compliant with the language specification (e.g. type system, goroutines, operations, built-ins, etc.).
 2. **[Go Standard Library](https://pkg.go.dev/std)**: mostly compatible. GopherJS attempts to support as much of standard library as possible, but certain functionality is impossible or difficult to implement within the JavaScript runtime, most of which is related to os interaction, low-level runtime manipulation or `unsafe`. See [package compatibility table](packages.md) and [syscall support](syscalls.md) for details.
 3. **Build system and tooling**: partially compatible. The `gopherjs` CLI tool is used to build and test GopherJS code. It currently supports building `GOPATH` projects, but Go Modules support is missing (see https://github.com/gopherjs/gopherjs/issues/855). Our goal is to reach complete feature parity with the `go` tool, but there is a large amount of work required to get there. Other notable challenges include:
    - Limited [compiler directive](pragma.md) (a.k.a. "pragma") support. Those are considered compiler implementation-specific and are generally not portable.
    - GopherJS ships with [standard library augmentations](../compiler/natives/src/), that are required to make it work in a browser. Those are applied on-the-fly during the build process and are generally invisible to any third-party tooling such as linters. In most cases that shouldn't matter, since they never change public interfaces of the standard library packages, but this is something to be aware of.
    - Runtime debuggers and profilers. Since GopherJS compiles Go to JavaScript, one must use JavaScript debuggers and profilers (e.g. browser dev tools) instead of the normal Go ones (e.g. delve or pprof). Unfortunately, limited sourcemap support makes this experience less than ideal at the moment.

## Go version compatibility

In general, for a given release of GopherJS the following statements _should_ be true:

  - GopherJS compiler can be built from source with the latest stable Go release at the time when the GopherJS release is created, or any newer Go release. 
  
    Example: you can build GopherJS `1.12-3` with Go `1.12` or newer.

  - GopherJS compiler can build code using standard library of a specific Go version, normally the latest stable at the time of GopherJS release. In most cases, it should be compatible with all patch versions within the minor Go version, but this is not guaranteed. 
  
    Example: GopherJS `1.16.0+go1.16.2` (see [developer documentation](https://github.com/gopherjs/gopherjs/wiki/Developer-Guidelines#versions) about GopherJS versioning schema) can build code with GOROOT pointing at Go `1.16.0` or `1.16.2`, but not at Go `1.15.x` or `1.17.x`.

  - Users can use older GopherJS releases if they need to target older Go versions, but only the latest GopherJS release is officially supported at this time.

_Note_: we would love to make GopherJS compatible with more Go releases, but the amount of effort required to support that exceeds amount of time we currently have available. If you wish to lend your help to make that possible, please reach out to us!

## How to report a incompatibility issue?

First of all, please check the list of known issues below, [package support table](packages.md), as well as [open issues](https://github.com/gopherjs/gopherjs/issues) on GitHub. If the issue is already known, great! You've saved yourself a bit of time. Feel free to add any extra details you think are relevant, though.

If the issue is not known yet, please open a new issue on GitHub and include the following information:

  1. Go and GopherJS versions you are using.
  2. In which environment do you see the issue (browser, nodejs, etc.).
  3. A minimal program that behaves differently when compiled with the regular Go compiler and GopherJS.

Now that the issue exists, we (GopherJS maintainers) will do our best to address it as promptly as we can. Note, however, that all of us are working on GopherJS in our spare time after our job and family responsibilities, so we can't guarantee an immediate fix. 

🚧 If you would like to help, please consider [submitting a pull request](https://github.com/gopherjs/gopherjs/wiki/Developer-Guidelines) with a fix. If you are unsure of the best way to approach the issue, we will be happy to share whatever knowledge we can! 😃

## How to write portable code

For the most part, GopherJS shouldn't require any special support for the code that only uses [supported standard packages](packages.md).

However, if you do need to provide different implementations depending on the target architecture, you can use [build constraints](https://golang.org/cmd/go/#hdr-Build_constraints) to do so:

  - `//+build js` — the source will be used for GopherJS and Go WebAssembly, but not for native builds.
  - `//+build js,-wasm` — the source will be used for GopherJS only, and not WebAssembly or native builds.
  - `//+build js,wasm` — the source will be used for Go WebAssembly, and not GopherJS or native builds.

Also be careful about using GopherJS-specific packages (e.g. `github.com/gopherjs/gopherjs/js`) or features (e.g. [wrapping JavaScript objects](https://github.com/gopherjs/gopherjs/wiki/JavaScript-Tips-and-Gotchas#tips) into Go structs), since those won't work outside of GopherJS.

### Portability between Go and TinyGo WebAssembly implementations

GopherJS implements `syscall/js` package, so it _should_ be able to run most code written for WebAssembly. However, in practice this topic is largely unexplored at this time.

It is worth noting that GopherJS emulates 32-bit environment, whereas Go WebAssembly is 64 bit, so you should use fixed-size types if you need to guarantee consistent behavior between the two architectures.

🚧 If you have first-hand experience with this, please consider adding it to this section!

## Known Go specification violations

  - Bit shifts of a negative amount (e.g. `42 << -1`) panic in Go, but not in GopherJS.
  - See also [open issues](https://github.com/gopherjs/gopherjs/issues) and [known failing compiler tests](https://github.com/gopherjs/gopherjs/blob/master/tests/run.go).