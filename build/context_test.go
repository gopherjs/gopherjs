package build

import (
	"fmt"
	"go/build"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/gopherjs/gopherjs/compiler/gopherjspkg"
	"golang.org/x/tools/go/buildutil"
)

func TestSimpleCtx(t *testing.T) {
	gopherjsRoot := filepath.Join(DefaultGOROOT, "src", "github.com", "gopherjs", "gopherjs")
	fs := &withPrefix{gopherjspkg.FS, gopherjsRoot}
	ec := embeddedCtx(fs, "", []string{})

	gc := goCtx("", []string{})

	t.Run("exists", func(t *testing.T) {
		tests := []struct {
			buildCtx XContext
			wantPkg  *PackageData
		}{
			{
				buildCtx: ec,
				wantPkg: &PackageData{
					Package:   expectedPackage(&ec.bctx, "github.com/gopherjs/gopherjs/js"),
					IsVirtual: true,
				},
			}, {
				buildCtx: gc,
				wantPkg: &PackageData{
					Package:   expectedPackage(&gc.bctx, "fmt"),
					IsVirtual: false,
				},
			},
		}

		for _, test := range tests {
			t.Run(fmt.Sprintf("%T", test.buildCtx), func(t *testing.T) {
				importPath := test.wantPkg.ImportPath
				got, err := test.buildCtx.Import(importPath, "", build.FindOnly)
				if err != nil {
					t.Fatalf("ec.Import(%q) returned error: %s. Want: no error.", importPath, err)
				}
				if diff := cmp.Diff(test.wantPkg, got, cmpopts.IgnoreUnexported(*got)); diff != "" {
					t.Errorf("ec.Import(%q) returned diff (-want,+got):\n%s", importPath, diff)
				}
			})
		}
	})

	t.Run("not found", func(t *testing.T) {
		tests := []struct {
			buildCtx   XContext
			importPath string
		}{
			{
				buildCtx:   ec,
				importPath: "package/not/found",
			}, {
				// Outside of the main module.
				buildCtx:   gc,
				importPath: "package/not/found",
			}, {
				// In the main module.
				buildCtx:   gc,
				importPath: "github.com/gopherjs/gopherjs/not/found",
			},
		}

		for _, test := range tests {
			t.Run(fmt.Sprintf("%T", test.buildCtx), func(t *testing.T) {
				_, err := ec.Import(test.importPath, "", build.FindOnly)
				want := "cannot find package"
				if err == nil || !strings.Contains(err.Error(), want) {
					t.Errorf("ec.Import(%q) returned error: %s. Want error containing %q.", test.importPath, err, want)
				}
			})
		}
	})
}

func TestChainedCtx(t *testing.T) {
	// Construct a chained context of two fake contexts so that we could verify
	// fallback behavior.
	cc := chainedCtx{
		primary: simpleCtx{
			bctx: *buildutil.FakeContext(map[string]map[string]string{
				"primaryonly": {"po.go": "package primaryonly"},
				"both":        {"both.go": "package both"},
			}),
			isVirtual: false,
		},
		secondary: simpleCtx{
			bctx: *buildutil.FakeContext(map[string]map[string]string{
				"both":          {"both_secondary.go": "package both"},
				"secondaryonly": {"so.go": "package secondaryonly"},
			}),
			isVirtual: true,
		},
	}

	tests := []struct {
		importPath      string
		wantFromPrimary bool
	}{
		{
			importPath:      "primaryonly",
			wantFromPrimary: true,
		}, {
			importPath:      "both",
			wantFromPrimary: true,
		}, {
			importPath:      "secondaryonly",
			wantFromPrimary: false,
		},
	}

	for _, test := range tests {
		t.Run(test.importPath, func(t *testing.T) {
			pkg, err := cc.Import(test.importPath, "", 0)
			if err != nil {
				t.Errorf("cc.Import() returned error: %v. Want: no error.", err)
			}
			gotFromPrimary := !pkg.IsVirtual
			if gotFromPrimary != test.wantFromPrimary {
				t.Errorf("Got package imported from primary: %t. Want: %t.", gotFromPrimary, test.wantFromPrimary)
			}
		})
	}
}

func expectedPackage(bctx *build.Context, importPath string) *build.Package {
	targetRoot := path.Clean(fmt.Sprintf("%s/pkg/%s_%s", bctx.GOROOT, bctx.GOOS, bctx.GOARCH))
	return &build.Package{
		Dir:           path.Join(bctx.GOROOT, "src", importPath),
		ImportPath:    importPath,
		Root:          bctx.GOROOT,
		SrcRoot:       path.Join(bctx.GOROOT, "src"),
		PkgRoot:       path.Join(bctx.GOROOT, "pkg"),
		PkgTargetRoot: targetRoot,
		BinDir:        path.Join(bctx.GOROOT, "bin"),
		Goroot:        true,
		PkgObj:        path.Join(targetRoot, importPath+".a"),
	}
}
