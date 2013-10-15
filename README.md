GopherJS - A transpiler from Go to JavaScript
---------------------------------------------

### What is supported?
GopherJS is able to turn itself (and all packages it uses) into pure JavaScript code that runs in all major browsers. This suggests a quite good coverage of Go's specification. However, there are some known exceptions listed below and some unknown exceptions that I would love to hear about when you find some.

### Not yet supported
Those features are not implemented yet, but on the roadmap:

- implicit panics (division by zero, etc.)
- exact runtime type assertions for compound types without a name
- reflection
- goroutines, channels, select
- goto

### Derivations from Go specification
Some tradeoffs have to be made in order to avoid huge performance impacts. Please get in contact if those are deal breakers for you.

- int32, uint32, int64 and uint64 have emulated overflow, all other integer types do not
- calls on nil cause a panic except for slice types

### Interface to external JavaScript
A function's body can be written in JavaScript by putting the code in a string constant with the name `js_[function name]` for package functions and `js_[type name]_[method name]` for methods. In that case, GopherJS disregards the Go function body and instead generates `function(...) { [constant's value] }`. This allows functions to have a Go signature that the type checker can use while being able to call external JavaScript functions.
