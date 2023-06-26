//go:build legacy_syscall

package syscall

import (
	"github.com/gopherjs/gopherjs/js"
)

var (
	syscallModule      *js.Object
	alreadyTriedToLoad = false
	minusOne           = -1
)

var warningPrinted = false

func printWarning() {
	if !warningPrinted {
		js.Global.Get("console").Call("error", "warning: system calls not available, see https://github.com/gopherjs/gopherjs/blob/master/doc/syscalls.md")
	}
	warningPrinted = true
}

func syscallByName(name string) *js.Object {
	defer recover() // return nil if recovered

	if syscallModule == nil {
		if alreadyTriedToLoad {
			return nil
		}
		alreadyTriedToLoad = true
		require := js.Global.Get("require")
		if require == js.Undefined {
			panic("")
		}
		syscallModule = require.Invoke("syscall")
	}
	return syscallModule.Get(name)
}

func Syscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno) {
	if f := syscallByName("Syscall"); f != nil {
		r := f.Invoke(trap, a1, a2, a3)
		return uintptr(r.Index(0).Int()), uintptr(r.Index(1).Int()), Errno(r.Index(2).Int())
	}
	printWarning()
	return uintptr(minusOne), 0, ENOSYS
}

func Syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno) {
	if f := syscallByName("Syscall6"); f != nil {
		r := f.Invoke(trap, a1, a2, a3, a4, a5, a6)
		return uintptr(r.Index(0).Int()), uintptr(r.Index(1).Int()), Errno(r.Index(2).Int())
	}
	if trap != 202 { // kern.osrelease on OS X, happens in init of "os" package
		printWarning()
	}
	return uintptr(minusOne), 0, ENOSYS
}

func RawSyscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno) {
	if f := syscallByName("Syscall"); f != nil {
		r := f.Invoke(trap, a1, a2, a3)
		return uintptr(r.Index(0).Int()), uintptr(r.Index(1).Int()), Errno(r.Index(2).Int())
	}
	printWarning()
	return uintptr(minusOne), 0, ENOSYS
}

func rawSyscallNoError(trap, a1, a2, a3 uintptr) (r1, r2 uintptr) {
	if f := syscallByName("Syscall"); f != nil {
		r := f.Invoke(trap, a1, a2, a3)
		return uintptr(r.Index(0).Int()), uintptr(r.Index(1).Int())
	}
	printWarning()
	return uintptr(minusOne), 0
}

func RawSyscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno) {
	if f := syscallByName("Syscall6"); f != nil {
		r := f.Invoke(trap, a1, a2, a3, a4, a5, a6)
		return uintptr(r.Index(0).Int()), uintptr(r.Index(1).Int()), Errno(r.Index(2).Int())
	}
	printWarning()
	return uintptr(minusOne), 0, ENOSYS
}
