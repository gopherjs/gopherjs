// +build js

package unix

const (
	randomTrap                = 0
	fstatatTrap               = 0
	getrandomTrap     uintptr = 0
	copyFileRangeTrap uintptr = 0
)

func IsNonblock(fd int) (nonblocking bool, err error) {
	return false, nil
}
