package build

import (
	"fmt"
	"path/filepath"
)

func mustAbs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		panic(fmt.Errorf("failed to get absolute path to %s", p))
	}
	return a
}
