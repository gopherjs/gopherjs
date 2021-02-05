// +build go1.16

package main

import (
	"go/build"
	"go/token"

	gbuild "github.com/goplusjs/gopherjs/build"
)

func makeTestPkg(pkg *gbuild.PackageData, xtest bool) *gbuild.PackageData {
	if xtest {
		return &gbuild.PackageData{
			Package: &build.Package{
				Name:            pkg.Name + "_test",
				ImportPath:      pkg.ImportPath + "_test",
				Dir:             pkg.Dir,
				GoFiles:         pkg.XTestGoFiles,
				Imports:         pkg.XTestImports,
				EmbedPatternPos: pkg.XTestEmbedPatternPos,
			},
			IsTest: true,
		}
	} else {
		pmap := make(map[string][]token.Position)
		for k, v := range pkg.EmbedPatternPos {
			pmap[k] = v
		}
		for k, v := range pkg.TestEmbedPatternPos {
			if ov, ok := pmap[k]; ok {
				pmap[k] = append(v, ov...)
			} else {
				pmap[k] = v
			}
		}
		return &gbuild.PackageData{
			Package: &build.Package{
				Name:            pkg.Name,
				ImportPath:      pkg.ImportPath,
				Dir:             pkg.Dir,
				GoFiles:         append(pkg.GoFiles, pkg.TestGoFiles...),
				Imports:         append(pkg.Imports, pkg.TestImports...),
				EmbedPatternPos: pmap,
			},
			IsTest:  true,
			JSFiles: pkg.JSFiles,
		}
	}
}
