package build

import (
	"bytes"
	"fmt"
	gobuild "go/build"
	"go/printer"
	"go/token"
	"strconv"
	"testing"

	"github.com/gopherjs/gopherjs/internal/srctesting"
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
	stdOnly := goCtx(DefaultEnv())
	// Skip post-load package tweaks, since we are interested in the complete set
	// of original sources.
	stdOnly.noPostTweaks = true
	// We only care about standard library, so skip all GOPATH packages.
	stdOnly.bctx.GOPATH = ""
	forward, _, err := importgraphutil.BuildNoTests(&stdOnly.bctx)
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
	populateImportSet := func(imports []string) stringSet {
		set := stringSet{}
		for _, p := range imports {
			set[p] = struct{}{}
			switch p {
			case "sync":
				set["github.com/gopherjs/gopherjs/nosync"] = struct{}{}
			}
			transitiveImports := forward.Search(p)
			for p := range transitiveImports {
				set[p] = struct{}{}
			}
		}
		return set
	}

	// Check all standard library packages.
	//
	// The general strategy is to first import each standard library package using the
	// normal build.Import, which returns a *build.Package. That contains Imports, TestImports,
	// and XTestImports values that are considered the "real imports".
	//
	// That list of direct imports is then expanded to the transitive closure by populateImportSet,
	// meaning all packages that are indirectly imported are also added to the set.
	//
	// Then, github.com/gopherjs/gopherjs/build.parseAndAugment(*build.Package) returns []*ast.File.
	// Those augmented parsed Go files of the package are checked, one file at at time, one import
	// at a time. Each import is verified to belong in the set of allowed real imports.
	matches, matchErr := stdOnly.Match([]string{"std"})
	if matchErr != nil {
		t.Fatalf("Failed to list standard library packages: %s", err)
	}
	for _, pkgName := range matches {
		pkgName := pkgName // Capture for the goroutine.
		t.Run(pkgName, func(t *testing.T) {
			t.Parallel()

			pkg, err := stdOnly.Import(pkgName, "", gobuild.ImportComment)
			if err != nil {
				t.Fatalf("gobuild.Import: %v", err)
			}

			for _, pkgVariant := range []*PackageData{pkg, pkg.TestPackage(), pkg.XTestPackage()} {
				t.Logf("Checking package %s...", pkgVariant)

				// Capture the set of unmodified package imports.
				realImports := populateImportSet(pkgVariant.Imports)

				// Use parseAndAugment to get a list of augmented AST files.
				fset := token.NewFileSet()
				files, _, err := parseAndAugment(stdOnly, pkgVariant, pkgVariant.IsTest, fset)
				if err != nil {
					t.Fatalf("github.com/gopherjs/gopherjs/build.parseAndAugment: %v", err)
				}

				// Verify imports of augmented AST files.
				for _, f := range files {
					fileName := fset.File(f.Pos()).Name()
					for _, imp := range f.Imports {
						importPath, err := strconv.Unquote(imp.Path.Value)
						if err != nil {
							t.Fatalf("strconv.Unquote(%v): %v", imp.Path.Value, err)
						}
						if importPath == "github.com/gopherjs/gopherjs/js" {
							continue
						}
						if _, ok := realImports[importPath]; !ok {
							t.Errorf("augmented package %q imports %q in file %v, but real %q doesn't:\nrealImports = %v",
								pkgVariant, importPath, fileName, pkgVariant.ImportPath, realImports)
						}
					}
				}
			}
		})
	}
}

// stringSet is used to print a set of strings in a more readable way.
type stringSet map[string]struct{}

func (m stringSet) String() string {
	s := make([]string, 0, len(m))
	for v := range m {
		s = append(s, v)
	}
	return fmt.Sprintf("%q", s)
}

func TestOverlayAugmentation(t *testing.T) {
	tests := []struct {
		desc    string
		src     string
		want    string
		expInfo map[string]overrideInfo
	}{
		{
			desc:    `prune an unused import`,
			src:     `import foo "some/other/bar"`,
			want:    ``,
			expInfo: map[string]overrideInfo{},
		}, {
			desc: `remove function`,
			src: `func Foo(a, b int) int {
					return a + b
				}`,
			want: `func Foo(a, b int) int {
				return a + b
			}`,
			expInfo: map[string]overrideInfo{
				`Foo`: {},
			},
		}, {
			desc: `keep function`,
			src: `//gopherjs:keep-original
				func Foo(a, b int) int {
					return a + b
				}`,
			want: `//gopherjs:keep-original
				func Foo(a, b int) int {
					return a + b
				}`,
			expInfo: map[string]overrideInfo{
				`Foo`: {keepOriginal: true},
			},
		}, {
			desc: `purge function`,
			src: `//gopherjs:purge
				func Foo(a, b int) int {
					return a + b
				}`,
			want: ``,
			expInfo: map[string]overrideInfo{
				`Foo`: {},
			},
		}, {
			desc: `purge struct removes an import`,
			src: `import "bytes"
				import "math"
			
				//gopherjs:purge
				type Foo struct {
					bar *bytes.Buffer
				}
				
				const Tau = math.Pi * 2.0`,
			want: `import "math"
					
				const Tau = math.Pi * 2.0`,
			expInfo: map[string]overrideInfo{
				`Foo`: {purgeMethods: true},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			pkgName := "package testpackage\n\n"
			fsetSrc := token.NewFileSet()
			fileSrc := srctesting.Parse(t, fsetSrc, pkgName+test.src)

			overrides := map[string]overrideInfo{}
			augmentOverlayFile(fsetSrc, fileSrc, overrides)
			pruneImports(fsetSrc, fileSrc)

			buf := &bytes.Buffer{}
			_ = printer.Fprint(buf, fsetSrc, fileSrc)
			got := buf.String()

			fsetWant := token.NewFileSet()
			fileWant := srctesting.Parse(t, fsetWant, pkgName+test.want)

			buf.Reset()
			_ = printer.Fprint(buf, fsetWant, fileWant)
			want := buf.String()

			if got != want {
				t.Errorf("augmentOverlayFile and pruneImports got unexpected code:\n"+
					"returned:\n\t%q\nwant:\n\t%q", got, want)
			}

			for key, expInfo := range test.expInfo {
				if gotInfo, ok := overrides[key]; !ok {
					t.Errorf(`%q was expected but not gotten`, key)
				} else if expInfo != gotInfo {
					t.Errorf(`%q had wrong info, got %+v`, key, gotInfo)
				}
			}
			for key, gotInfo := range overrides {
				if _, ok := test.expInfo[key]; !ok {
					t.Errorf(`%q with %+v was not expected`, key, gotInfo)
				}
			}
		})
	}

}
