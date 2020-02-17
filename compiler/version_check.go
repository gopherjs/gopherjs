// +build go1.12

package compiler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

// Version is the GopherJS compiler version string.
const Version = "1.12-3"

// GoVersion is the current Go 1.x version that GopherJS is compatible with.
const GoVersion = 12

// CheckGoVersion checks the version of the Go distribution
// at goroot, and reports an error if it's not compatible
// with this version of the GopherJS compiler.
func CheckGoVersion(goroot string) error {
	v, err := ioutil.ReadFile(filepath.Join(goroot, "VERSION"))
	if err != nil {
		return fmt.Errorf("GopherJS %s requires a Go 1.12.x distribution, but failed to read its VERSION file: %v", Version, err)
	}
	if !bytes.HasPrefix(v, []byte("go1.12")) { // TODO(dmitshur): Change this before Go 1.120 comes out.
		return fmt.Errorf("GopherJS %s requires a Go 1.12.x distribution, but found version %s", Version, v)
	}
	return nil
}
