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
const Version = "1.16.4+go1.16.7"

// GoVersion is the current Go 1.x version that GopherJS is compatible with.
const GoVersion = 16

// CheckGoVersion checks the version of the Go distribution
// at goroot, and reports an error if it's not compatible
// with this version of the GopherJS compiler.
func CheckGoVersion(goroot string) error {
	if nvc, err := strconv.ParseBool(os.Getenv("GOPHERJS_SKIP_VERSION_CHECK")); err == nil && nvc {
		return nil
	}
	v, err := ioutil.ReadFile(filepath.Join(goroot, "VERSION"))
	if err != nil {
		return fmt.Errorf("GopherJS %s requires a Go 1.16.x distribution, but failed to read its VERSION file: %v", Version, err)
	}
	if !bytes.HasPrefix(v, []byte("go1.16")) {
		return fmt.Errorf("GopherJS %s requires a Go 1.16.x distribution, but found version %s", Version, v)
	}
	return nil
}
