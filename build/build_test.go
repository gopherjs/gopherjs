package build

import (
	gobuild "go/build"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/kisielk/gotool"
	"github.com/shurcooL/go/importgraphutil"
)

// Natives augment the standard library with GopherJS-specific changes.
// This test ensures that none of the standard library packages are modified
// in a way that adds imports which the original upstream standard library package
// does not already import. Doing that can increase generated output size or cause
// other unexpected issues (since the cmd/go tool does not know about these extra imports),
// so it's best to avoid it.
//
// It checks all standard library packages. Each package is considered as a normal
// package, as a test package, and as an external test package.
func TestNativesDontImportExtraPackages(t *testing.T) {
	// Calculate the forward import graph for all standard library packages.
	// It's needed for populateImportSet.
	stdOnly := gobuild.Default
	stdOnly.GOPATH = "" // We only care about standard library, so skip all GOPATH packages.
	forward, _, err := importgraphutil.BuildNoTests(&stdOnly)
	if err != nil {
		t.Fatalf("importgraphutil.BuildNoTests: %v", err)
	}

	// populateImportSet takes a slice of imports, and populates set with those
	// imports, as well as their transitive dependencies. That way, the set can
	// be quickly queried to check if a package is in the import graph of imports.
	//
	// Note, this does not include transitive imports of test/xtest packages,
	// which could cause some false positives. It currently doesn't, but if it does,
	// then support for that should be added here.
	populateImportSet := func(imports []string, set *map[string]struct{}) {
		for _, p := range imports {
			(*set)[p] = struct{}{}
			switch p {
			case "sync":
				(*set)["github.com/gopherjs/gopherjs/nosync"] = struct{}{}
			}
			transitiveImports := forward.Search(p)
			for p := range transitiveImports {
				(*set)[p] = struct{}{}
			}
		}
	}

	// Check all standard library packages.
	for _, pkg := range gotool.ImportPaths([]string{"std"}) {
		// Normal package.
		{
			bpkg, err := gobuild.Import(pkg, "", gobuild.ImportComment)
			if err != nil {
				t.Fatalf("gobuild.Import: %v", err)
			}
			realImports := make(map[string]struct{})
			populateImportSet(bpkg.Imports, &realImports)

			fset := token.NewFileSet()
			files, err := parse(bpkg, false, fset)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			for _, f := range files {
				fileName := fset.File(f.Pos()).Name()
				normalFile := !strings.HasSuffix(fileName, "_test.go")
				if !normalFile {
					continue
				}
				for _, imp := range f.Imports {
					importPath, err := strconv.Unquote(imp.Path.Value)
					if err != nil {
						t.Fatalf("strconv.Unquote(%v): %v", imp.Path.Value, err)
					}
					if importPath == "github.com/gopherjs/gopherjs/js" {
						continue
					}
					if _, ok := realImports[importPath]; !ok {
						t.Errorf("augmented normal package %q imports %q in file %v, but real %q doesn't:\nrealImports = %q", bpkg.ImportPath, importPath, fileName, bpkg.ImportPath, toSlice(realImports))
					}
				}
			}
		}

		// Test package.
		{
			bpkg, err := gobuild.Import(pkg, "", gobuild.ImportComment)
			if err != nil {
				t.Fatalf("gobuild.Import: %v", err)
			}
			realTestImports := make(map[string]struct{})
			populateImportSet(bpkg.TestImports, &realTestImports)

			fset := token.NewFileSet()
			files, err := parse(bpkg, true, fset)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			for _, f := range files {
				fileName, pkgName := fset.File(f.Pos()).Name(), f.Name.String()
				testFile := strings.HasSuffix(fileName, "_test.go") && !strings.HasSuffix(pkgName, "_test")
				if !testFile {
					continue
				}
				for _, imp := range f.Imports {
					importPath, err := strconv.Unquote(imp.Path.Value)
					if err != nil {
						t.Fatalf("strconv.Unquote(%v): %v", imp.Path.Value, err)
					}
					if importPath == "github.com/gopherjs/gopherjs/js" {
						continue
					}
					if _, ok := realTestImports[importPath]; !ok {
						t.Errorf("augmented test package %q imports %q in file %v, but real %q doesn't:\nrealTestImports = %q", bpkg.ImportPath, importPath, fileName, bpkg.ImportPath, toSlice(realTestImports))
					}
				}
			}
		}

		// External test package.
		{
			bpkg, err := gobuild.Import(pkg, "", gobuild.ImportComment)
			if err != nil {
				t.Fatalf("gobuild.Import: %v", err)
			}
			realXTestImports := make(map[string]struct{})
			populateImportSet(bpkg.XTestImports, &realXTestImports)

			fset := token.NewFileSet()
			files, err := parse(bpkg, true, fset)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			for _, f := range files {
				fileName, pkgName := fset.File(f.Pos()).Name(), f.Name.String()
				xTestFile := strings.HasSuffix(fileName, "_test.go") && strings.HasSuffix(pkgName, "_test")
				if !xTestFile {
					continue
				}
				for _, imp := range f.Imports {
					importPath, err := strconv.Unquote(imp.Path.Value)
					if err != nil {
						t.Fatalf("strconv.Unquote(%v): %v", imp.Path.Value, err)
					}
					if importPath == "github.com/gopherjs/gopherjs/js" {
						continue
					}
					if _, ok := realXTestImports[importPath]; !ok {
						t.Errorf("augmented external test package %q imports %q in file %v, but real %q doesn't:\nrealXTestImports = %q", bpkg.ImportPath, importPath, fileName, bpkg.ImportPath, toSlice(realXTestImports))
					}
				}
			}
		}
	}
}

func toSlice(m map[string]struct{}) []string {
	s := make([]string, 0, len(m))
	for v := range m {
		s = append(s, v)
	}
	return s
}
