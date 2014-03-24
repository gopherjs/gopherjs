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
Get or update GopherJS and dependencies with:
```
go get -u github.com/gopherjs/gopherjs
```
Now you can use  `./bin/gopherjs build [files]` or `./bin/gopherjs install [package]` which behave similar to the `go` tool. For `main` packages, these commands create a `.js` file and `.js.map` source map in the current directory or in `$GOPATH/bin`. The generated JavaScript file can be used as usual in a website. Go's `println` builtin prints to the JavaScript console via `console.log`. If you want to run the generated code with Node.js, see [this page](doc/nodejs.md).

*Note: GopherJS will try to write compiled object files of the core packages to your $GOROOT/pkg directory. If that fails, it will fall back to $GOPATH/pkg.*

### Getting started
#### 1. Interacting directly with the DOM
Attach listeners in the `main` method by using the `js` package:
```go
package main

import (
  "github.com/gopherjs/gopherjs/js"
)

func main() {
  nameElement := js.Global.Get("document").Call("getElementById", "name")
  greetingElement := js.Global.Get("document").Call("getElementById", "greeting")
  nameElement.Call("addEventListener", "input", func() {
    greetingElement.Set("innerText", "Hello "+nameElement.Get("value").String()+"!")
  })
}
```
The HTML code looks like this:
```html
<html>
  <head>
    <meta charset="utf-8">
  </head>
  <body>
    Your name: <input type="text" id="name"></input>
    <h1 id="greeting"></h1>
    <script src="test.js"></script>
  </body>
</html>
```
#### 2. Providing library functions for use in other JavaScript code
Set a global variable to a map that contains the functions:
```go
package main
 
import (
  "github.com/gopherjs/gopherjs/js"
  "github.com/rolaveric/gopherjs/user"
)
 
func main() {
  js.Global.Set("user", map[string]interface{}{
    "registerDB": user.RegisterDB,
    "new":        user.New,
    "get":        user.Get,
    "all":        user.All,
  })
}
```
This example if from [Jason Stone's blog post](http://legacytotheedge.blogspot.de/2014/03/gopherjs-go-to-javascript-transpiler.html) about GopherJS. Take a look for further details.

### Interface to native JavaScript
The package `github.com/gopherjs/gopherjs/js` provides functions for interacting with native JavaScript APIs. Please see its [documentation](http://godoc.org/github.com/gopherjs/gopherjs/js) for further details.

### Community
- Get help in the [Google Group](https://groups.google.com/d/forum/gopherjs)
- See the list of [bindings to JavaScript APIs and libraries](https://github.com/gopherjs/gopherjs/wiki/bindings) by community members
- Follow [GopherJS on Twitter](https://twitter.com/GopherJS)

### Roadmap
These features are not implemented yet, but on the roadmap:

- goroutines, channels, select
- output minification
- float32 and complex64 currently have the same precision as float64 and complex128
