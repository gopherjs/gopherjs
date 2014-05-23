package jslib

import (
	"github.com/metakeule/gopherjs/build"
)

func BuildDir(packagePath string, options *build.Options) error {
	options.Normalize()
	s := build.NewSession(options)
	return s.BuildDir(packagePath, "main", "")
}
