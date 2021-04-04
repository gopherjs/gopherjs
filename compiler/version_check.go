// +build go1.16

package compiler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

// Version is the GopherJS compiler version string.
const Version = "1.16.0+go1.16.2"

// GoVersion is the current Go 1.x version that GopherJS is compatible with.
const GoVersion = 16

// CheckGoVersion checks the version of the Go distribution
// at goroot, and reports an error if it's not compatible
// with this version of the GopherJS compiler.
func CheckGoVersion(goroot string) error {
	v, err := ioutil.ReadFile(filepath.Join(goroot, "VERSION"))
	if err != nil {
		return fmt.Errorf("GopherJS %s requires a Go 1.16.x distribution, but failed to read its VERSION file: %v", Version, err)
	}
	if !bytes.HasPrefix(v, []byte("go1.16")) {
		return fmt.Errorf("GopherJS %s requires a Go 1.16.x distribution, but found version %s", Version, v)
	}
	return nil
}
