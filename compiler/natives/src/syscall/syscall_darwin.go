// +build js

package syscall

import (
	"github.com/gopherjs/gopherjs/js"
)

func funcPC(f func()) uintptr {
	switch js.InternalObject(f) {
	case js.InternalObject(libc_getgroups_trampoline):
		return SYS_GETGROUPS
	case js.InternalObject(libc_setgroups_trampoline):
		return SYS_SETGROUPS
	case js.InternalObject(libc_wait4_trampoline):
		return SYS_WAIT4
	case js.InternalObject(libc_accept_trampoline):
		return SYS_ACCEPT
	case js.InternalObject(libc_bind_trampoline):
		return SYS_BIND
	case js.InternalObject(libc_connect_trampoline):
		return SYS_CONNECT
	case js.InternalObject(libc_socket_trampoline):
		return SYS_SOCKET
	case js.InternalObject(libc_getsockopt_trampoline):
		return SYS_GETSOCKOPT
	case js.InternalObject(libc_setsockopt_trampoline):
		return SYS_SETSOCKOPT
	case js.InternalObject(libc_getpeername_trampoline):
		return SYS_GETPEERNAME
	case js.InternalObject(libc_getsockname_trampoline):
		return SYS_GETSOCKNAME
	case js.InternalObject(libc_shutdown_trampoline):
		return SYS_SHUTDOWN
	case js.InternalObject(libc_socketpair_trampoline):
		return SYS_SOCKETPAIR
	case js.InternalObject(libc_recvfrom_trampoline):
		return SYS_RECVFROM
	case js.InternalObject(libc_sendto_trampoline):
		return SYS_SENDTO
	case js.InternalObject(libc_recvmsg_trampoline):
		return SYS_RECVMSG
	case js.InternalObject(libc_sendmsg_trampoline):
		return SYS_SENDMSG
	case js.InternalObject(libc_kevent_trampoline):
		return SYS_KEVENT
	case js.InternalObject(libc_utimes_trampoline):
		return SYS_UTIMES
	case js.InternalObject(libc_futimes_trampoline):
		return SYS_FUTIMES
	case js.InternalObject(libc_fcntl_trampoline):
		return SYS_FCNTL
	case js.InternalObject(libc_pipe_trampoline):
		return SYS_PIPE
	case js.InternalObject(libc_kill_trampoline):
		return SYS_KILL
	case js.InternalObject(libc_access_trampoline):
		return SYS_ACCESS
	case js.InternalObject(libc_adjtime_trampoline):
		return SYS_ADJTIME
	case js.InternalObject(libc_chdir_trampoline):
		return SYS_CHDIR
	case js.InternalObject(libc_chflags_trampoline):
		return SYS_CHFLAGS
	case js.InternalObject(libc_chmod_trampoline):
		return SYS_CHMOD
	case js.InternalObject(libc_chown_trampoline):
		return SYS_CHOWN
	case js.InternalObject(libc_chroot_trampoline):
		return SYS_CHROOT
	case js.InternalObject(libc_close_trampoline):
		return SYS_CLOSE
	case js.InternalObject(libc_dup_trampoline):
		return SYS_DUP
	case js.InternalObject(libc_dup2_trampoline):
		return SYS_DUP2
	case js.InternalObject(libc_exchangedata_trampoline):
		return SYS_EXCHANGEDATA
	case js.InternalObject(libc_fchdir_trampoline):
		return SYS_FCHDIR
	case js.InternalObject(libc_fchflags_trampoline):
		return SYS_FCHFLAGS
	case js.InternalObject(libc_fchmod_trampoline):
		return SYS_FCHMOD
	case js.InternalObject(libc_fchown_trampoline):
		return SYS_FCHOWN
	case js.InternalObject(libc_flock_trampoline):
		return SYS_FLOCK
	case js.InternalObject(libc_fpathconf_trampoline):
		return SYS_FPATHCONF
	case js.InternalObject(libc_fsync_trampoline):
		return SYS_FSYNC
	case js.InternalObject(libc_ftruncate_trampoline):
		return SYS_FTRUNCATE
	case js.InternalObject(libc_getdtablesize_trampoline):
		return SYS_GETDTABLESIZE
	case js.InternalObject(libc_getegid_trampoline):
		return SYS_GETEGID
	case js.InternalObject(libc_geteuid_trampoline):
		return SYS_GETEUID
	case js.InternalObject(libc_getgid_trampoline):
		return SYS_GETGID
	case js.InternalObject(libc_getpgid_trampoline):
		return SYS_GETPGID
	case js.InternalObject(libc_getpgrp_trampoline):
		return SYS_GETPGRP
	case js.InternalObject(libc_getpid_trampoline):
		return SYS_GETPID
	case js.InternalObject(libc_getppid_trampoline):
		return SYS_GETPPID
	case js.InternalObject(libc_getpriority_trampoline):
		return SYS_GETPRIORITY
	case js.InternalObject(libc_getrlimit_trampoline):
		return SYS_GETRLIMIT
	case js.InternalObject(libc_getrusage_trampoline):
		return SYS_GETRUSAGE
	case js.InternalObject(libc_getsid_trampoline):
		return SYS_GETSID
	case js.InternalObject(libc_getuid_trampoline):
		return SYS_GETUID
	case js.InternalObject(libc_issetugid_trampoline):
		return SYS_ISSETUGID
	case js.InternalObject(libc_kqueue_trampoline):
		return SYS_KQUEUE
	case js.InternalObject(libc_lchown_trampoline):
		return SYS_LCHOWN
	case js.InternalObject(libc_link_trampoline):
		return SYS_LINK
	case js.InternalObject(libc_listen_trampoline):
		return SYS_LISTEN
	case js.InternalObject(libc_mkdir_trampoline):
		return SYS_MKDIR
	case js.InternalObject(libc_mkfifo_trampoline):
		return SYS_MKFIFO
	case js.InternalObject(libc_mknod_trampoline):
		return SYS_MKNOD
	case js.InternalObject(libc_mlock_trampoline):
		return SYS_MLOCK
	case js.InternalObject(libc_mlockall_trampoline):
		return SYS_MLOCKALL
	case js.InternalObject(libc_mprotect_trampoline):
		return SYS_MPROTECT
	case js.InternalObject(libc_munlock_trampoline):
		return SYS_MUNLOCK
	case js.InternalObject(libc_munlockall_trampoline):
		return SYS_MUNLOCKALL
	case js.InternalObject(libc_open_trampoline):
		return SYS_OPEN
	case js.InternalObject(libc_pathconf_trampoline):
		return SYS_PATHCONF
	case js.InternalObject(libc_pread_trampoline):
		return SYS_PREAD
	case js.InternalObject(libc_pwrite_trampoline):
		return SYS_PWRITE
	case js.InternalObject(libc_read_trampoline):
		return SYS_READ
	case js.InternalObject(libc_readlink_trampoline):
		return SYS_READLINK
	case js.InternalObject(libc_rename_trampoline):
		return SYS_RENAME
	case js.InternalObject(libc_revoke_trampoline):
		return SYS_REVOKE
	case js.InternalObject(libc_rmdir_trampoline):
		return SYS_RMDIR
	case js.InternalObject(libc_lseek_trampoline):
		return SYS_LSEEK
	case js.InternalObject(libc_select_trampoline):
		return SYS_SELECT
	case js.InternalObject(libc_setegid_trampoline):
		return SYS_SETEGID
	case js.InternalObject(libc_seteuid_trampoline):
		return SYS_SETEUID
	case js.InternalObject(libc_setgid_trampoline):
		return SYS_SETGID
	case js.InternalObject(libc_setlogin_trampoline):
		return SYS_SETLOGIN
	case js.InternalObject(libc_setpgid_trampoline):
		return SYS_SETPGID
	case js.InternalObject(libc_setpriority_trampoline):
		return SYS_SETPRIORITY
	case js.InternalObject(libc_setprivexec_trampoline):
		return SYS_SETPRIVEXEC
	case js.InternalObject(libc_setregid_trampoline):
		return SYS_SETREGID
	case js.InternalObject(libc_setreuid_trampoline):
		return SYS_SETREUID
	case js.InternalObject(libc_setrlimit_trampoline):
		return SYS_SETRLIMIT
	case js.InternalObject(libc_setsid_trampoline):
		return SYS_SETSID
	case js.InternalObject(libc_settimeofday_trampoline):
		return SYS_SETTIMEOFDAY
	case js.InternalObject(libc_setuid_trampoline):
		return SYS_SETUID
	case js.InternalObject(libc_symlink_trampoline):
		return SYS_SYMLINK
	case js.InternalObject(libc_sync_trampoline):
		return SYS_SYNC
	case js.InternalObject(libc_truncate_trampoline):
		return SYS_TRUNCATE
	case js.InternalObject(libc_umask_trampoline):
		return SYS_UMASK
	case js.InternalObject(libc_undelete_trampoline):
		return SYS_UNDELETE
	case js.InternalObject(libc_unlink_trampoline):
		return SYS_UNLINK
	case js.InternalObject(libc_unmount_trampoline):
		return SYS_UNMOUNT
	case js.InternalObject(libc_write_trampoline):
		return SYS_WRITE
	case js.InternalObject(libc_writev_trampoline):
		return SYS_WRITEV
	case js.InternalObject(libc_mmap_trampoline):
		return SYS_MMAP
	case js.InternalObject(libc_munmap_trampoline):
		return SYS_MUNMAP
	case js.InternalObject(libc_fork_trampoline):
		return SYS_FORK
	case js.InternalObject(libc_ioctl_trampoline):
		return SYS_IOCTL
	case js.InternalObject(libc_execve_trampoline):
		return SYS_EXECVE
	case js.InternalObject(libc_exit_trampoline):
		return SYS_EXIT
	case js.InternalObject(libc_fstat64_trampoline):
		return SYS_FSTAT64
	case js.InternalObject(libc_fstatfs64_trampoline):
		return SYS_FSTATFS64
	case js.InternalObject(libc_gettimeofday_trampoline):
		return SYS_GETTIMEOFDAY
	case js.InternalObject(libc_lstat64_trampoline):
		return SYS_LSTAT64
	case js.InternalObject(libc_stat64_trampoline):
		return SYS_STAT64
	case js.InternalObject(libc_statfs64_trampoline):
		return SYS_STATFS64
	case js.InternalObject(libc_ptrace_trampoline):
		return SYS_PTRACE
	}
	return uintptr(minusOne)
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
	return Syscall6(trap, a1, a2, a3, a4, a5, a6)
}

func rawSyscall(trap, a1, a2, a3 uintptr) (r1, r2 uintptr, err Errno) {
	return RawSyscall(trap, a1, a2, a3)
}

func rawSyscall6(trap, a1, a2, a3, a4, a5, a6 uintptr) (r1, r2 uintptr, err Errno) {
	return RawSyscall6(trap, a1, a2, a3, a4, a5, a6)
}
