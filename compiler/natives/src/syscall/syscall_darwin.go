// +build js

package syscall

func funcPC(func()) uintptr {
	return 0
}

func syscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno) {
	printWarning()
	return uintptr(minusOne), 0, EACCES
}

func syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno) {
	printWarning()
	return uintptr(minusOne), 0, EACCES
}

func syscall6X(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno) {
	printWarning()
	return uintptr(minusOne), 0, EACCES
}

func rawSyscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno) {
	printWarning()
	return uintptr(minusOne), 0, EACCES
}

func rawSyscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno) {
	printWarning()
	return uintptr(minusOne), 0, EACCES
}
