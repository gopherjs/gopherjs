GopherJS - A transpiler from Go to JavaScript
---------------------------------------------

[![Build Status](https://travis-ci.org/gopherjs/gopherjs.png?branch=master)](https://travis-ci.org/gopherjs/gopherjs)

GopherJS translates Go code ([golang.org](http://golang.org/)) to pure JavaScript code. Its main purpose is to give you the opportunity to write front-end code in Go which will still run in all browsers. Give GopherJS a try on the [GopherJS Playground](http://gopherjs.github.io/playground/).

You can take advantage of Go's elegant type system and other compile-time checks that can have a huge impact on bug detection and the ability to refactor, especially for big projects. Just think of how often a JavaScript method has extra handling of some legacy parameter scheme, because you don't know exactly if some other code is still calling it in that old way or not. GopherJS will tell you and if it does not complain, you can be sure that this kind of bug is not present any more.

### Design Goals
- performance of generated code (see [HTML5 game engine benchmark](http://ajhager.github.io/enj/) by Joseph Hager)
- similarity between Go code and generated JavaScript code for easier debugging
- compatibility with existing libraries (see the list of [bindings to JavaScript APIs and libraries](https://github.com/gopherjs/gopherjs/wiki/bindings))
- small size of generated code

### What is supported?
In one sentence: Everything except goroutines. Yes, I know that you want goroutines and I am working heavily on them. But hey, it is still better to write Go with callbacks than JavaScript with callbacks, right? A lot of Go's packages do already work, see the [compatibility table](doc/packages.md). If you want this still missing feature, please consider to support this project with a star to show your interest.

### Installation and Usage
Get GopherJS and dependencies with:
```
go get github.com/gopherjs/gopherjs
```
Now you can use  `./bin/gopherjs build` and `./bin/gopherjs install` which behave similar to the `go` tool. The generated JavaScript files can be used as usual in a website. Go's `println` builtin prints to the JavaScript console via `console.log`. If you want to run the generated code with Node.js, see [this page](doc/nodejs.md).

*Note: GopherJS will try to write compiled object files of the core packages to your $GOROOT/pkg directory. If that fails, it will fall back to $GOPATH/pkg.*

### Interface to native JavaScript
The package `github.com/gopherjs/gopherjs/js` provides functions for interacting with native JavaScript APIs. Please see its [documentation](http://godoc.org/github.com/gopherjs/gopherjs/js) for further details.

### Community
- Get help in the [Google Group](https://groups.google.com/d/forum/gopherjs)
- See the list of [bindings to JavaScript APIs and libraries](https://github.com/gopherjs/gopherjs/wiki/bindings) by community members

### Roadmap
These features are not implemented yet, but on the roadmap:

- goroutines, channels, select
- output minification
- source maps
- float32 and complex64 currently have the same precision as float64 and complex128
