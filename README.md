GopherJS - A transpiler from Go to JavaScript
---------------------------------------------

### Installation and Usage
Get GopherJS and dependencies with: 
```
go get github.com/neelance/gopherjs
```
Patch go/types and compile GopherJS again (will become optional soon):
```
patch -p 1 -d src/code.google.com/p/go.tools/ < src/github.com/neelance/gopherjs/patches/go.types.patch
go install github.com/neelance/gopherjs
```
Use `./bin/gopherjs` similar to the `go` tool.

### What is supported?
GopherJS is able to turn itself (and all packages it uses) into pure JavaScript code that runs in all major browsers. This suggests a quite good coverage of Go's specification. However, there are some known exceptions listed below and some unknown exceptions that I would love to hear about when you find them.

### Roadmap
These features are not implemented yet, but on the roadmap:

- implicit panics (division by zero, etc.)
- complex numbers
- reflection (already partially done)
- exact runtime type assertions for compound types without a name
- goroutines, channels, select
- goto
- output minification
- source maps

### Deviations from Go specification
Some tradeoffs have to be made in order to avoid huge performance impacts. Please get in contact if those are deal breakers for you.

- int, uint and uintptr do not overflow
- calls on nil cause a panic except for slice types

### Interface to external JavaScript
A function's body can be written in JavaScript by putting the code in a string constant with the name `js_[function name]` for package functions and `js_[type name]_[method name]` for methods. In that case, GopherJS disregards the Go function body and instead generates `function(...) { [constant's value] }`. This allows functions to have a Go signature that the type checker can use while being able to call external JavaScript functions.
