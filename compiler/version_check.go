//go:build go1.16
// +build go1.16

package compiler

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

// Version is the GopherJS compiler version string.
const Version = "1.17.0+go1.17"

// GoVersion is the current Go 1.x version that GopherJS is compatible with.
const GoVersion = 17

// CheckGoVersion checks the version of the Go distribution
// at goroot, and reports an error if it's not compatible
// with this version of the GopherJS compiler.
func CheckGoVersion(goroot string) error {
	if nvc, err := strconv.ParseBool(os.Getenv("GOPHERJS_SKIP_VERSION_CHECK")); err == nil && nvc {
		return nil
	}
	v, err := ioutil.ReadFile(filepath.Join(goroot, "VERSION"))
	if err != nil {
		return fmt.Errorf("GopherJS %s requires a Go 1.%d.x distribution, but failed to read its VERSION file: %v", Version, GoVersion, err)
	}
	if !bytes.HasPrefix(v, []byte("go1."+strconv.Itoa(GoVersion))) {
		return fmt.Errorf("GopherJS %s requires a Go 1.%d.x distribution, but found version %s", Version, GoVersion, v)
	}
	return nil
}
