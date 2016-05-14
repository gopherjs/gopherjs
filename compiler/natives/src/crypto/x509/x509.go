// +build js

package x509

import "os"

func initSystemRoots() {
	// no system roots
}

func execSecurityRoots() (*CertPool, error) {
	return nil, os.ErrNotExist
}
