// +build js

package unix

import "syscall"

const randomTrap = 0
const fstatatTrap = 0

func IsNonblock(fd int) (nonblocking bool, err error) {
	return false, nil
}

func unlinkat(dirfd int, path string, flags int) error {
	// There's no SYS_UNLINKAT defined in Go 1.12 for Darwin,
	// so just implement unlinkat using unlink for now.
	return syscall.Unlink(path)
}
