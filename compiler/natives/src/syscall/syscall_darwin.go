// +build js

package syscall

import "github.com/gopherjs/gopherjs/js"

func funcPC(f func()) uintptr {
	switch js.InternalObject(f) {
	case js.InternalObject(libc_open_trampoline):
		return SYS_OPEN
	case js.InternalObject(libc_stat64_trampoline):
		return SYS_STAT64
	case js.InternalObject(libc_fstat64_trampoline):
		return SYS_FSTAT64
	case js.InternalObject(libc_lstat64_trampoline):
		return SYS_LSTAT64
	case js.InternalObject(libc_mkdir_trampoline):
		return SYS_MKDIR
	case js.InternalObject(libc_chdir_trampoline):
		return SYS_CHDIR
	case js.InternalObject(libc_rmdir_trampoline):
		return SYS_RMDIR
	case js.InternalObject(libc___getdirentries64_trampoline):
		return SYS_GETDIRENTRIES64
	case js.InternalObject(libc_getattrlist_trampoline):
		return SYS_GETATTRLIST
	case js.InternalObject(libc_symlink_trampoline):
		return SYS_SYMLINK
	case js.InternalObject(libc_readlink_trampoline):
		return SYS_READLINK
	case js.InternalObject(libc_fcntl_trampoline):
		return SYS_FCNTL
	case js.InternalObject(libc_read_trampoline):
		return SYS_READ
	case js.InternalObject(libc_pread_trampoline):
		return SYS_PREAD
	case js.InternalObject(libc_write_trampoline):
		return SYS_WRITE
	case js.InternalObject(libc_lseek_trampoline):
		return SYS_LSEEK
	case js.InternalObject(libc_close_trampoline):
		return SYS_CLOSE
	case js.InternalObject(libc_unlink_trampoline):
		return SYS_UNLINK
	case js.InternalObject(libc_getpid_trampoline):
		return SYS_GETPID
	case js.InternalObject(libc_getuid_trampoline):
		return SYS_GETUID
	case js.InternalObject(libc_getgid_trampoline):
		return SYS_GETGID
	default:
		// If we just return -1, the caller can only print an unhelpful generic error message, like
		// "signal: bad system call".
		// So, execute f() to get a more helpful error message that includes the syscall name, like
		// "runtime error: native function not implemented: syscall.libc_getpid_trampoline".
		f()
		return uintptr(minusOne)
	}
}

func syscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno) {
	return Syscall(trap, a1, a2, a3)
}

func syscallX(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno) {
	return Syscall(trap, a1, a2, a3)
}

func syscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno) {
	return Syscall6(trap, a1, a2, a3, a4, a5, a6)
}

func syscall6X(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno) {
	panic("syscall6X is not implemented")
}

func rawSyscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno) {
	return RawSyscall(trap, a1, a2, a3)
}

func rawSyscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno) {
	return RawSyscall6(trap, a1, a2, a3, a4, a5, a6)
}
