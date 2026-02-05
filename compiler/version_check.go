//go:build go1.20

package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Version is the GopherJS compiler version string.
const Version = "1.20.1+go1.20.14"

// GoVersion is the current Go 1.x version that GopherJS is compatible with.
const GoVersion = 20

// CheckGoVersion checks the version of the Go distribution
// at goroot, and reports an error if it's not compatible
// with this version of the GopherJS compiler.
func CheckGoVersion(goroot string) error {
	if nvc, err := strconv.ParseBool(os.Getenv("GOPHERJS_SKIP_VERSION_CHECK")); err == nil && nvc {
		return nil
	}
	v, err := goRootVersion(goroot)
	if err != nil {
		return fmt.Errorf("unable to detect Go version for %q: %w", goroot, err)
	}
	if !strings.HasPrefix(v, "go1."+strconv.Itoa(GoVersion)) {
		return fmt.Errorf("GopherJS %s requires a Go 1.%d.x distribution, but found version %s", Version, GoVersion, v)
	}
	return nil
}

// goRootVersion determines the Go release for the given GOROOT installation.
func goRootVersion(goroot string) (string, error) {
	if b, err := os.ReadFile(filepath.Join(goroot, "VERSION")); err == nil {
		// Standard Go distribution has a VERSION file inside its GOROOT,
		// checking its first line is the most efficient option.
		v, _, _ := strings.Cut(string(b), "\n")
		return v, nil
	}

	// Fall back to the "go version" command.
	cmd := exec.Command(filepath.Join(goroot, "bin", "go"), "version")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("`go version` command failed: %w", err)
	}
	// Expected output: go version go1.20.14 linux/amd64
	parts := strings.Split(string(out), " ")
	if len(parts) != 4 {
		return "", fmt.Errorf("unexpected `go version` output %q, expected 4 words", string(out))
	}
	return parts[2], nil
}

// GoRelease does a best-effort to identify the Go release we are building with.
// If unable to determine the precise version for the given GOROOT, falls back
// to the best guess available.
func GoRelease(goroot string) string {
	v, err := goRootVersion(goroot)
	if err == nil {
		// Prefer using the actual version of the GOROOT we are working with.
		return v
	}

	// Use Go version GopherJS release was tested against as a fallback. By
	// convention, it is included in the GopherJS version after the plus sign.
	parts := strings.Split(Version, "+")
	if len(parts) == 2 {
		return parts[1]
	}

	// If everything else fails, return just the Go version without patch level.
	return fmt.Sprintf("go1.%d", GoVersion)
}
