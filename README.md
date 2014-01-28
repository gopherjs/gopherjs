GopherJS - A transpiler from Go to JavaScript
---------------------------------------------

### Useful Links
- Try GopherJS on the [GopherJS Playground](http://neelance.github.io/gopherjs-playground/)
- Enjoy the speed of the [HTML5 game engine benchmark](http://ajhager.github.io/enj/) (by Joseph Hager)
- Get help in the [Google Group](https://groups.google.com/d/forum/gopherjs)
- See the list of [bindings to JavaScript APIs and libraries](https://github.com/neelance/gopherjs/wiki/bindings)

### What is GopherJS?
GopherJS translates [Go code](http://golang.org/) to pure JavaScript code. Its main purpose is to give you the opportunity to write front-end code in Go which will still run in all browsers.

You can take advantage of Go's elegant type system and other compile-time checks that can have a huge impact on bug detection and the ability to refactor, especially for big projects. Just think of how often a JavaScript method has extra handling of some legacy parameter scheme, because you don't know exactly if some other code is still calling it in that old way or not. GopherJS will tell you and if it does not complain, you can be sure that this kind of bug is not present any more.

### What is supported?
- interface to native JavaScript code ([see below](#interface-to-native-javascript))
- all basic types, including 64-bit integers and complex numbers
- arrays, slices, maps and structures
- full type system with support for interfaces and type assertions
- reflection for all types
- many packages have been successfully tested, see [compatibility table](doc/packages.md)

### Installation and Usage
Get GopherJS and dependencies with: 
```
go get github.com/neelance/gopherjs
```
Now you can use  `./bin/gopherjs build` and `./bin/gopherjs install` which behave similar to the `go` tool. The generated JavaScript files can be used as usual in a website. Go's `println` builtin prints to the JavaScript console via `console.log`.

*Note: GopherJS will try to write compiled object files of the core packages to your $GOROOT/pkg directory. If that fails, it will fall back to $GOPATH/pkg.*

### Node.js
You can also run the generated code with Node.js instead of a browser. However, system calls (e.g. writing to the console via the `fmt` package or most of the `os` functions) will not work until you compile and install the syscall module. If you just need console output, you can use `println` instead.
The syscall module currently only supports OS X. Please tell me if you would like to have support for other operating systems. On OS X, get the latest Node.js 0.11 release from [here](http://blog.nodejs.org/release/) or via `brew install node --devel`. Then compile and install the module:
```
npm install --global node-gyp
cd src/github.com/neelance/gopherjs/node-syscall/
node-gyp rebuild
mkdir -p ~/.node_libraries/
cp build/Release/syscall.node ~/.node_libraries/syscall.node
cd ../../../../../
```

### Interface to native JavaScript
The package `github.com/neelance/gopherjs/js` ([documentation](js/js.go)) provides functions for interacting with native JavaScript APIs. Calls to these functions are treated specially by GopherJS and translated directly to their JavaScript syntax. Type conversions between Go types and JavaScript types are performed automatically according to the table below. Types not listed are passed through. The second column denotes the types that are used when converting to `interface{}`.

| Go types                       | Go interface type              | JavaScript type |
| ------------------------------ | ------------------------------ | --------------- |
| bool                           | bool                           | Boolean         |
| int?, uint?, float?            | float64                        | Number          |
| string                         | string                         | String          |
| [?]int8                        | []int8                         | Int8Array       |
| [?]int16                       | []int16                        | Int16Array      |
| [?]int32, [?]int               | []int                          | Int32Array      |
| [?]uint8                       | []uint8                        | Int8Array       |
| [?]uint16                      | []uint16                       | Int16Array      |
| [?]uint32, [?]uint, [?]uintptr | []uint                         | Int32Array      |
| [?]float32                     | []float32                      | Float32Array    |
| [?]float64                     | []float64                      | Float64Array    |
| all other slices and arrays    | []interface{}                  | Array           |
| maps                           | map[string]interface{}         | Object          |
| functions                      | func(...interface{}) js.Object | Function        |
| time.Time                      | time.Time                      | Date            |

### Roadmap
These features are not implemented yet, but on the roadmap:

- goto
- goroutines, channels, select
- output minification
- source maps
- float32 and complex64 currently have the same precision as float64 and complex128

[![Analytics](https://ga-beacon.appspot.com/UA-46799660-1/gopherjs/README.md)](https://github.com/igrigorik/ga-beacon)
