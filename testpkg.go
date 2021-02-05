// +build !go1.16

package main

import (
	"go/build"

	gbuild "github.com/goplusjs/gopherjs/build"
)

func makeTestPkg(pkg *gbuild.PackageData, xtest bool) *gbuild.PackageData {
	if xtest {
		return &gbuild.PackageData{
			Package: &build.Package{
				Name:       pkg.Name + "_test",
				ImportPath: pkg.ImportPath + "_test",
				Dir:        pkg.Dir,
				GoFiles:    pkg.XTestGoFiles,
				Imports:    pkg.XTestImports,
			},
			IsTest: true,
		}
	} else {
		return &gbuild.PackageData{
			Package: &build.Package{
				Name:       pkg.Name,
				ImportPath: pkg.ImportPath,
				Dir:        pkg.Dir,
				GoFiles:    append(pkg.GoFiles, pkg.TestGoFiles...),
				Imports:    append(pkg.Imports, pkg.TestImports...),
			},
			IsTest:  true,
			JSFiles: pkg.JSFiles,
		}
	}
}
