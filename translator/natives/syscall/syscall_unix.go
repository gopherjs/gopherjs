// +build js,!windows

package syscall

import (
	"github.com/gopherjs/gopherjs/js"
	"unsafe"
)

func init() {
	process := js.Global.Get("process")
	if !process.IsUndefined() {
		jsEnv := process.Get("env")
		envkeys := js.Global.Get("Object").Call("keys", jsEnv)
		envs = make([]string, envkeys.Length())
		for i := 0; i < envkeys.Length(); i++ {
			key := envkeys.Index(i).Str()
			envs[i] = key + "=" + jsEnv.Get(key).Str()
		}
	}
}

var syscallModule js.Object

func syscall(name string) js.Object {
	if syscallModule == nil {
		require := js.Global.Get("require")
		if require.IsUndefined() {
			syscallHandler := js.Global.Get("$syscall")
			if !syscallHandler.IsUndefined() {
				return syscallHandler
			}
			panic("syscalls not available")
		}
		syscallModule = require.Invoke("syscall")
	}
	return syscallModule.Get(name)
}

func Syscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno) {
	r := syscall("Syscall").Invoke(trap, a1, a2, a3)
	return uintptr(r.Index(0).Int()), uintptr(r.Index(1).Int()), Errno(r.Index(2).Int())
}

func Syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno) {
	r := syscall("Syscall6").Invoke(trap, a1, a2, a3, a4, a5, a6)
	return uintptr(r.Index(0).Int()), uintptr(r.Index(1).Int()), Errno(r.Index(2).Int())
}

func RawSyscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno) {
	r := syscall("Syscall").Invoke(trap, a1, a2, a3)
	return uintptr(r.Index(0).Int()), uintptr(r.Index(1).Int()), Errno(r.Index(2).Int())
}

func RawSyscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno) {
	r := syscall("Syscall6").Invoke(trap, a1, a2, a3, a4, a5, a6)
	return uintptr(r.Index(0).Int()), uintptr(r.Index(1).Int()), Errno(r.Index(2).Int())
}

func BytePtrFromString(s string) (*byte, error) {
	return (*byte)(unsafe.Pointer(js.Global.Call("go$stringToBytes", s, true).Unsafe())), nil
}
