Using GopherJS with Node.js
---------------------------

You can run the generated code with Node.js instead of a browser. However, system calls (e.g. writing to the console via the `fmt` package or most of the `os` functions) will not work until you compile and install the syscall module. If you just need console output, you can use `println` instead.
The syscall module currently only supports OS X. Please tell me if you would like to have support for other operating systems. On OS X, get the latest Node.js 0.11 release from [here](http://blog.nodejs.org/release/) or via `brew install node --devel`. Then compile and install the module:
```
npm install --global node-gyp
cd src/github.com/neelance/gopherjs/node-syscall/
node-gyp rebuild
mkdir -p ~/.node_libraries/
cp build/Release/syscall.node ~/.node_libraries/syscall.node
cd ../../../../../
```
