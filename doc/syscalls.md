## System Calls

System calls are the bridge between your application and your operating system. They are used whenever you access something outside of your application's memory, for example when you write to the console, when you read or write files or when you access the network. In Go, system calls are mostly used by the `os` package, hence the name. When using GopherJS you need to consider if system calls are available or not.

Starting with 1.18, GopherJS provides to the same [set of cross-platform](https://pkg.go.dev/syscall?GOOS=js) syscalls as standard Go WebAssembly, emulating them via JavaScript APIs available in the runtime (browser or Node.js).

### Output redirection to console

If system calls are not available in your environment (see below), then a special redirection of `os.Stdout` and `os.Stderr` is applied. It buffers a line until it is terminated by a line break and then prints it via JavaScript's `console.log` to your browser's JavaScript console or your system console. That way, `fmt.Println` etc. work as expected, even if system calls are not available.

### In Browser

The JavaScript environment of a web browser is completely isolated from your operating system to protect your machine. You don't want any web page to read or write files on your disk without your consent. That is why system calls are not and will never be available when running your code in a web browser.

However, certain subsets of syscalls can be emulated using third-party libraries. For example, [BrowserFS](https://github.com/jvilk/BrowserFS) library can be used to emulate Node.js file system API in a browser using HTML5 LocalStorage or other fallbacks.

### Node.js on all platforms

GopherJS emulates syscalls for accessing file system (and a few others) using Node.js standard [`fs`](https://nodejs.org/api/fs.html) and [`process`](https://nodejs.org/api/process.html) APIs. No additional extensions are required for this in GopherJS 1.18 and newer.

### Node.js with the legacy node-syscall extension.

Prior to 1.18 GopherJS required a custom Node extension to be installed that provided access to system calls on Linux and MacOS. Currently this extension is deprecated and its support will be removed entirely in a future release. This decision is motivated by several factors:

- This extension has been developed before Go had WebAssembly support and at the time there was no easier way to provide file system access. Today standard library for `js/wasm` provides most of the relevant functionality without the need for custom extensions.
- It required GopherJS to support building Go standard library with multiple different GOOS/GOARCH combinations, which significantly increased maintenance effort and slowed down support for new Go versions. It was not supported on Windows entirely.
- Using this extension required non-trivial setup for the users who needed file system access.
- The extension itself contained significant technical debt and potential memory leaks.
- File system syscalls use asynchronous Node.js API, so other goroutines doesn't get blocked.

Issue [#693](https://github.com/gopherjs/gopherjs/issues/693) has a detailed discussion of this.

In GopherJS 1.18 support for this extension is disabled by default to reduce the output size. It can be enabled with a build tag `legacy_syscall` (for example, `gopherjs build --tags legacy_syscall pkg/name`) with the following caveats:

- `node-syscall` extension must be built and installed according to instructions below.
- Functions `syscall.Syscall`, `syscall.Syscall6`, `syscall.RawSyscall` and `syscall.RawSyscall6` will be changed to use the extension API and can be called from the third-party code.
- Standard library is still built for `js/wasm` regardless of the host OS, so the syscall package API will remain reduced compared to `linux` or `darwin`.
- All functions in the `syscall` package that GopherJS emulates via Node.js APIs will continue using those APIs.
- While executing a legacy syscall, all goroutines get blocked. This may cause some programs not to behave as expected.

We strongly recommend upgrading your package to not use `syscall` package directly and use higher-level APIs in the `os` package, which will continue working.

The module is compatible with Node.js version 10.0.0 (or newer). If you want to use an older version you can opt to not install the module, but then system calls are not available.

Compile and install the module with:

```
cd gopherjs/node-syscall/
npm install
```

You can copy build/Release/syscall.node into you `node_modules` directory and run `node -r syscall` to make sure the module can be loaded successfully.

Alternatively, in _your_ `package.json` you can do something like this:

```
{
  "dependencies": {
    "syscall": "file:path/to/gopherjs/node-syscall"
  }
}
```

Which will make `npm install` in your project capable of building the extension. You may need to set `export NODE_PATH="$(npm root)"` to ensure that node can load modules from any working directory, for example when running `gopherjs test`.
