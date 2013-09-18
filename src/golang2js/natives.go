package main

var natives = map[string]string{
	"runtime": `
    sizeof_C_MStats = 3696;
    getgoroot = function() { return process.env["GOROOT"] || ""; };
    SetFinalizer = function() {};
  `,

	"sync/atomic": `
    AddInt32 = AddInt64 = AddUint32 = AddUint64 = AddUintptr = function(addr, delta) {
      var value = addr.get() + delta;
      addr.set(value);
      return value;
    };
    CompareAndSwapInt32 = CompareAndSwapInt64 = CompareAndSwapUint32 = CompareAndSwapUint64 = CompareAndSwapUintptr = function(addr, oldVal, newVal) {
      if (addr.get() === oldVal) {
        addr.set(newVal);
        return true;
      }
      return false;
    };
    StoreInt32 = StoreInt64 = StoreUint32 = StoreUint64 = StoreUintptr = function(addr, val) {
      addr.set(val);
    };
    LoadInt32 = LoadInt64 = LoadUint32 = LoadUint64 = LoadUintptr = function(addr) {
      return addr.get();
    };
  `,

	"syscall": `
    Exit = process.exit;
    Getenv = function(key) {
      var value = process.env[key];
      if (value === undefined) {
        return ["", false];
      }
      return [value, true];
    };
    Write = function(fd, p) {
      process.stdout.write(String.fromCharCode.apply(null, p.toArray()));
      return [p.length, null]
    };
  `,
}
