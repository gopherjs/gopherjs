package build

import (
	"go/build"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDirCacheMatchesImport(t *testing.T) {
	bctx := goCtx(DefaultEnv()).bctx
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		importPath string
		srcDir     string
		mode       build.ImportMode
	}{
		{name: "main module package", importPath: "github.com/gopherjs/gopherjs/compiler"},
		{name: "dependency", importPath: "github.com/spf13/cobra"},
		{name: "transitive dependency", importPath: "github.com/spf13/pflag"},
		{name: "FindOnly", importPath: "golang.org/x/tools/go/buildutil", mode: build.FindOnly},
		{name: "with srcDir", importPath: "github.com/google/go-cmp/cmp", srcDir: wd},
		{name: "missing package in main module", importPath: "github.com/gopherjs/gopherjs/does/not/exist"},
		{name: "missing module", importPath: "example.com/does/not/exist"},
	}

	dc := newDirCache()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if !resolvesWithGoList(bctx, test.importPath, test.srcDir, test.mode) {
				t.Fatalf("resolvesWithGoList(%q) = false, want true", test.importPath)
			}
			want, wantErr := bctx.Import(test.importPath, test.srcDir, test.mode)
			got, gotErr := dc.importPackage(bctx, test.importPath, test.srcDir, test.mode)

			if errString(gotErr) != errString(wantErr) {
				t.Errorf("importPackage(%q) returned error %q, want %q", test.importPath, errString(gotErr), errString(wantErr))
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("importPackage(%q) returned diff (-want,+got):\n%s", test.importPath, diff)
			}
		})
	}

	if dc.entries["github.com/spf13/pflag"] == nil {
		t.Errorf("pflag was not recorded when listing cobra's dependencies")
	}
}

func TestResolvesWithGoList(t *testing.T) {
	bctx := goCtx(DefaultEnv()).bctx
	const modulePkg = "github.com/spf13/cobra"

	withCallback := bctx
	withCallback.JoinPath = filepath.Join
	oldReleaseTags := bctx
	oldReleaseTags.ReleaseTags = bctx.ReleaseTags[:len(bctx.ReleaseTags)-1]
	withDir := bctx
	withDir.Dir = os.TempDir()

	tests := []struct {
		name       string
		bctx       build.Context
		importPath string
		srcDir     string
		mode       build.ImportMode
		env        map[string]string
		want       bool
	}{
		{name: "module package", bctx: bctx, importPath: modulePkg, want: true},
		{name: "standard library", bctx: bctx, importPath: "fmt"},
		{name: "local import", bctx: bctx, importPath: "./foo", srcDir: os.TempDir()},
		{name: "absolute import", bctx: bctx, importPath: "/foo"},
		{name: "IgnoreVendor", bctx: bctx, importPath: modulePkg, mode: build.IgnoreVendor},
		{name: "AllowBinary", bctx: bctx, importPath: modulePkg, mode: build.AllowBinary},
		{name: "file system callback", bctx: withCallback, importPath: modulePkg},
		{name: "non-default release tags", bctx: oldReleaseTags, importPath: modulePkg},
		{name: "srcDir in GOROOT", bctx: bctx, importPath: modulePkg, srcDir: filepath.Join(bctx.GOROOT, "src", "net")},
		{name: "relative srcDir with Dir set", bctx: withDir, importPath: modulePkg, srcDir: "foo"},
		{name: "GO111MODULE=off", bctx: bctx, importPath: modulePkg, env: map[string]string{"GO111MODULE": "off"}},
		{name: "GO111MODULE=auto", bctx: bctx, importPath: modulePkg, env: map[string]string{"GO111MODULE": "auto"}},
		{name: "GOPHERJS_NO_DIRCACHE", bctx: bctx, importPath: modulePkg, env: map[string]string{"GOPHERJS_NO_DIRCACHE": "1"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.env {
				t.Setenv(k, v)
			}
			if got := resolvesWithGoList(test.bctx, test.importPath, test.srcDir, test.mode); got != test.want {
				t.Errorf("resolvesWithGoList(%q, %q) = %v, want %v", test.importPath, test.srcDir, got, test.want)
			}
		})
	}
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
