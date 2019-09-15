package compiler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

// Version is the GopherJS compiler version string.
const Version = "1.12-2"

// CheckGoVersion checks the version of the Go distribution
// at goroot, and reports an error if it's not compatible
// with this version of the GopherJS compiler.
func CheckGoVersion(goroot string) error {
	v, err := ioutil.ReadFile(filepath.Join(goroot, "VERSION"))
	if err != nil {
		return fmt.Errorf("GopherJS %s requires a Go 1.12.x distribution, but failed to read its VERSION file: %v", Version, err)
	}
	if !bytes.Equal(v, []byte("go1.12")) && !bytes.HasPrefix(v, []byte("go1.12.")) {
		return fmt.Errorf("GopherJS %s requires a Go 1.12.x distribution, but found version %s", Version, v)
	}
	return nil
}
