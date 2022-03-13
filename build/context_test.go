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
	e := DefaultEnv()

	gopherjsRoot := filepath.Join(e.GOROOT, "src", "github.com", "gopherjs", "gopherjs")
	fs := &withPrefix{gopherjspkg.FS, gopherjsRoot}
	ec := embeddedCtx(fs, e)

	gc := goCtx(e)

	t.Run("exists", func(t *testing.T) {
		tests := []struct {
			buildCtx XContext
			wantPkg  *PackageData
		}{
			{
				buildCtx: ec,
				wantPkg: &PackageData{
					Package:   expectedPackage(&ec.bctx, "github.com/gopherjs/gopherjs/js", "wasm"),
					IsVirtual: true,
				},
			}, {
				buildCtx: gc,
				wantPkg: &PackageData{
					Package:   expectedPackage(&gc.bctx, "fmt", "wasm"),
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

func TestIsStd(t *testing.T) {
	realGOROOT := goCtx(DefaultEnv())
	overlayGOROOT := overlayCtx(DefaultEnv())
	gopherjsPackages := gopherjsCtx(DefaultEnv())
	tests := []struct {
		descr      string
		importPath string
		context    *simpleCtx
		want       bool
	}{
		{
			descr:      "real goroot, standard package",
			importPath: "fmt",
			context:    realGOROOT,
			want:       true,
		},
		{
			descr:      "real goroot, non-standard package",
			importPath: "github.com/gopherjs/gopherjs/build",
			context:    realGOROOT,
			want:       false,
		},
		{
			descr:      "real goroot, non-exiting package",
			importPath: "does/not/exist",
			context:    realGOROOT,
			want:       false,
		},
		{
			descr:      "overlay goroot, standard package",
			importPath: "fmt",
			context:    overlayGOROOT,
			want:       true,
		},
		{
			descr:      "embedded gopherjs packages, gopherjs/js package",
			importPath: "github.com/gopherjs/gopherjs/js",
			context:    gopherjsPackages,
			// When user's source tree doesn't contain gopherjs package (e.g. it uses
			// syscall/js API only), we pretend that gopherjs/js package is included
			// in the standard library.
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.descr, func(t *testing.T) {
			got := test.context.isStd(test.importPath, "")
			if got != test.want {
				t.Errorf("Got: simpleCtx.isStd(%q) = %v. Want: %v", test.importPath, got, test.want)
			}
		})
	}
}

func expectedPackage(bctx *build.Context, importPath string, goarch string) *build.Package {
	targetRoot := path.Clean(fmt.Sprintf("%s/pkg/%s_%s", bctx.GOROOT, bctx.GOOS, goarch))
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
