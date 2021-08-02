package build

import (
	"fmt"
	"os"
	"path/filepath"
)

func mustAbs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		panic(fmt.Errorf("failed to get absolute path to %s", p))
	}
	return a
}

// makeWritable attempts to make the given path writable by its owner.
func makeWritable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	err = os.Chmod(path, info.Mode()|0700)
	if err != nil {
		return err
	}
	return nil
}
