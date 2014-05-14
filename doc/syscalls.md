System Calls
------------

System calls are the bridge between your application and your operating system. They are used whenever you access something outside of your application's memory, for example when you write to the console, when you read or write files or when you access the network. In Go, system calls are mostly used by the `os` package, hence the name. When using GopherJS you need to consider if system calls are available or not.

### No fmt.Println?

If system calls are not available in your environment (see below), keep in mind that instead of using `fmt.Println` you can still use Go's `println` built-in to write to your browser's JavaScript console or your system console.

### In Browser

The JavaScript environment of a web browser is completely isolated from your operating system to protect your machine. You don't want any web page to read or write files on your disk without your consent. That is why system calls are not and will never be available when running your code in a web browser.

### Node.js on Windows

When running your code with Node.js on Windows, it is theoretically possible to use system calls. To do so, you would need a special Node.js module that provides direct access to system calls. However, since the interface is quite different from the one used on OS X and Linux, the system calls module included in GopherJS currently does not support Windows. Sorry. Get in contact if you feel like you want to change this situation.

### Node.js on OS X and Linux

GopherJS has support for system calls on OS X and Linux. Before running your code with Node.js, you need to install the system calls module. The module is only compatible with Node.js' unstable 0.11 releases, so if you want to stay with the stable version you can opt to not install the module, but then system calls are not available.

Install Node.js 0.11 with the Node Version Manager...
```
curl https://raw.githubusercontent.com/creationix/nvm/master/install.sh | bash
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
