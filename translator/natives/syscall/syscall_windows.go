// +build js

package syscall

func Syscall(trap, nargs, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno) {
	panic("syscalls not available on Windows")
}

func Syscall6(trap, nargs, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno) {
	panic("syscalls not available on Windows")
}

func Syscall9(trap, nargs, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) (r1, r2 uintptr, err Errno) {
	panic("syscalls not available on Windows")
}

func Syscall12(trap, nargs, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12 uintptr) (r1, r2 uintptr, err Errno) {
	panic("syscalls not available on Windows")
}

func Syscall15(trap, nargs, a1, a2, a3, a4, a5, a6, a7, a8, a9, a10, a11, a12, a13, a14, a15 uintptr) (r1, r2 uintptr, err Errno) {
	panic("syscalls not available on Windows")
}

func loadlibrary(filename *uint16) (handle uintptr, err Errno) {
	panic("syscalls not available on Windows")
}

func getprocaddress(handle uintptr, procname *uint8) (proc uintptr, err Errno) {
	panic("syscalls not available on Windows")
}

func getStdHandle(h int) (fd Handle) {
	return 0
}

func GetCommandLine() (cmd *uint16) {
	return
}

type DummyError struct{}

func (e DummyError) Error() string {
	return ""
}

func CommandLineToArgv(cmd *uint16, argc *int32) (argv *[8192]*[8192]uint16, err error) {
	return nil, DummyError{}
}

func Getenv(key string) (value string, found bool) {
	return "", false
}

func GetTimeZoneInformation(tzi *Timezoneinformation) (rc uint32, err error) {
	return 0, DummyError{}
}
