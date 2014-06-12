GopherJS - A compiler from Go to JavaScript
---------------------------------------------

### Building GopherJS currently fails because of a change in the go.tools repository that is affecting a lot of Go projects: 
The relevant issue is https://code.google.com/p/go/issues/detail?id=8191 (give it a star if you like to see it resolved soon). An easy workaround is to replace the `Float32Val` function in `src/code.google.com/p/go.tools/go/exact/exact.go` with the following code:

```go
func Float32Val(x Value) (float32, bool) {
	switch x := x.(type) {
	case int64Val:
		f := float32(x)
		return f, int64Val(f) == x
	case intVal:
		f, exact := new(big.Rat).SetFrac(x.val, int1).Float64()
		return float32(f), exact
	case floatVal:
		f, exact := x.val.Float64()
		return float32(f), exact
	case unknownVal:
		return 0, false
	}
	panic(fmt.Sprintf("%v not a Float", x))
}
```

[![Build Status](https://travis-ci.org/gopherjs/gopherjs.png?branch=master)](https://travis-ci.org/gopherjs/gopherjs)

GopherJS compiles Go code ([golang.org](http://golang.org/)) to pure JavaScript code. Its main purpose is to give you the opportunity to write front-end code in Go which will still run in all browsers. Give GopherJS a try on the [GopherJS Playground](http://gopherjs.github.io/playground/).

You can take advantage of Go's elegant type system and other compile-time checks that can have a huge impact on bug detection and the ability to refactor, especially for big projects. Just think of how often a JavaScript method has extra handling of some legacy parameter scheme, because you don't know exactly if some other code is still calling it in that old way or not. GopherJS will tell you and if it does not complain, you can be sure that this kind of bug is not present any more.

### Design Goals
- performance of generated code (see [HTML5 game engine benchmark](http://ajhager.github.io/enj/) by Joseph Hager)
- similarity between Go code and generated JavaScript code for easier debugging
- compatibility with existing libraries (see the list of [bindings to JavaScript APIs and libraries](https://github.com/gopherjs/gopherjs/wiki/bindings))
- small size of generated code

### What is supported?
In one sentence: Everything except goroutines. Yes, I know that you want goroutines and I am working heavily on them. But hey, it is still better to write Go with callbacks than JavaScript with callbacks, right? A lot of Go's packages do already work, see the [compatibility table](doc/packages.md). If you want this still missing feature, please consider to support this project with a star to show your interest.

### Installation and Usage
Get or update GopherJS and dependencies with:
```
go get -u github.com/gopherjs/gopherjs
```
Now you can use  `./bin/gopherjs build [files]` or `./bin/gopherjs install [package]` which behave similar to the `go` tool. For `main` packages, these commands create a `.js` file and `.js.map` source map in the current directory or in `$GOPATH/bin`. The generated JavaScript file can be used as usual in a website. Use `./bin/gopherjs help [command]` to get a list of possible command line flags, e.g. for minification and automatically watching for changes. If you want to run the generated code with Node.js, see [this page](doc/syscalls.md).

*Note: GopherJS will try to write compiled object files of the core packages to your $GOROOT/pkg directory. If that fails, it will fall back to $GOPATH/pkg.*

### Getting started
#### 1. Interacting with the DOM
Accessing the DOM directly via the `js` package (see below) is possible, but using a JavaScript framework is more elegant. Take a look at the [TodoMVC Example](https://github.com/gopherjs/todomvc) which is using the [jQuery bindings](https://github.com/gopherjs/jquery) or alternatively the [AngularJS bindings](https://github.com/gopherjs/go-angularjs). Additionally, there is a list of [bindings to JavaScript APIs and libraries](https://github.com/gopherjs/gopherjs/wiki/bindings) by community members.

#### 2. Providing library functions for use in other JavaScript code
Set a global variable to a map that contains the functions:
```go
package main

import "github.com/gopherjs/gopherjs/js"

func main() {
  js.Global.Set("myLibrary", map[string]interface{}{
    "someFunction": someFunction,
  })
}

func someFunction() {
  [...]
}
```
For more details see [Jason Stone's blog post](http://legacytotheedge.blogspot.de/2014/03/gopherjs-go-to-javascript-transpiler.html) about GopherJS.

### Interface to native JavaScript
The package `github.com/gopherjs/gopherjs/js` provides functions for interacting with native JavaScript APIs. Please see its [documentation](http://godoc.org/github.com/gopherjs/gopherjs/js) for further details.

### Community
- Get help in the [Google Group](https://groups.google.com/d/forum/gopherjs)
- See the list of [bindings to JavaScript APIs and libraries](https://github.com/gopherjs/gopherjs/wiki/bindings) by community members
- Follow [GopherJS on Twitter](https://twitter.com/GopherJS)

### Roadmap
These features are not implemented yet, but on the roadmap:

- goroutines, channels, select
- float32 and complex64 currently have the same precision as float64 and complex128
