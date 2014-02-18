Using GopherJS with Node.js
---------------------------

You can run the generated code with Node.js instead of a browser. However, system calls (e.g. writing to the console via the `fmt` package or most of the `os` functions) will not work until you compile and install the syscall module. If you just need console output, you can use `println` instead. The syscall module works on Linux and OS X.

Install Node.js 0.11 with the Node Version Manager...
```
curl https://raw.github.com/creationix/nvm/master/install.sh | bash
. $HOME/.nvm/nvm.sh
nvm install 0.11
nvm use 0.11
```
... or Homebrew.
```
brew install node --devel
```
Then compile and install the module:
```
npm install --global node-gyp
cd src/github.com/gopherjs/gopherjs/node-syscall/
node-gyp rebuild
mkdir -p ~/.node_libraries/
cp build/Release/syscall.node ~/.node_libraries/syscall.node
cd ../../../../../
```
