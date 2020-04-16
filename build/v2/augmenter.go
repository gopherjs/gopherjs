package build

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"io"
	"os"
	"path"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/natives"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/packages"
)

// SymbolFilter implements top-level symbol pruning for augmented packages.
//
// GopherJS provides custom implementations of some parts of standard library, which are added
// on top of the original stdlib sources. To avoid compilation errors due to duplicate symbols,
// SymbolFilter can Collect top-level symbols from overlay sources and then Prune conflicting
// symbols from the original sources.
type SymbolFilter map[string]bool

func (sf SymbolFilter) funcName(d *ast.FuncDecl) string {
	if d.Recv == nil || len(d.Recv.List) == 0 {
		return d.Name.Name
	}
	recv := d.Recv.List[0].Type
	if star, ok := recv.(*ast.StarExpr); ok {
		recv = star.X
	}
	return recv.(*ast.Ident).Name + "." + d.Name.Name
}

// traverse top-level symbols within the file and prune top-level symbols for which keep() returned
// false.
//
// This function is functionally very similar to ast.FilterFile with two differences: it doesn't
// descend into interface methods and struct fields, and it preserves imports.
func (sf SymbolFilter) traverse(f *ast.File, keep func(name string) bool) {
	astutil.Apply(f, func(c *astutil.Cursor) bool {
		switch d := c.Node().(type) {
		case *ast.File: // Root node.
			return true
		case *ast.FuncDecl: // Child of *ast.File.
			if !keep(sf.funcName(d)) {
				c.Delete()
			}
		case *ast.GenDecl: // Child of *ast.File.
			return c.Name() == "Decls"
		case *ast.ValueSpec: // Child of *ast.GenDecl.
			for i, name := range d.Names {
				if !keep(name.Name) {
					// Deleting variable/const declarations is somewhat fiddly (need to keep many different
					// slices inside of *ast.ValueSpec in sync), so we simply rename it to "_", which will
					// later be ignored.
					d.Names[i] = ast.NewIdent("_")
				}
			}
		case *ast.TypeSpec: // Child of *ast.GenDecl.
			if !keep(d.Name.Name) {
				c.Delete()
			}
		}
		return false
	}, nil)
}

// Collect names of top-level symbols in the source file. Doesn't modify the file itself.
func (sf SymbolFilter) Collect(f *ast.File) {
	sf.traverse(f, func(name string) bool {
		sf[name] = true
		return true
	})
}

// Prune in-place top-level symbols with names that match previously collected.
func (sf SymbolFilter) Prune(f *ast.File) {
	sf.traverse(f, func(name string) bool {
		return !sf[name]
	})
}

type Augmenter struct {
	natives *build.Context
}

func NewAugmenter() *Augmenter {
	a := Augmenter{}

	a.natives = &build.Context{
		GOROOT:   "/",
		GOOS:     build.Default.GOOS,
		GOARCH:   "js",
		Compiler: "gc",
		JoinPath: path.Join,
		SplitPathList: func(list string) []string {
			if list == "" {
				return nil
			}
			return strings.Split(list, "/")
		},
		IsAbsPath: path.IsAbs,
		IsDir: func(name string) bool {
			dir, err := natives.FS.Open(name)
			if err != nil {
				return false
			}
			defer dir.Close()
			info, err := dir.Stat()
			if err != nil {
				return false
			}
			return info.IsDir()
		},
		HasSubdir: func(root, name string) (rel string, ok bool) {
			panic("not implemented")
		},
		ReadDir: func(name string) (fi []os.FileInfo, err error) {
			dir, err := natives.FS.Open(name)
			if err != nil {
				return nil, err
			}
			defer dir.Close()
			return dir.Readdir(0)
		},
		OpenFile: func(name string) (r io.ReadCloser, err error) {
			return natives.FS.Open(name)
		},
	}

	return &a
}

func (a *Augmenter) Augment(pkg *packages.Package) error {
	augments, err := a.natives.Import(pkg.PkgPath, "", 0)
	if err != nil {
		// This is probably not an augmented package.
		// TODO: Make a more robust check.
		return nil
	}
	// TODO: Handle test and x-test packages.
	sources := augments.GoFiles

	symbolFilter := SymbolFilter{}

	extraAST := []*ast.File{}
	extraFiles := []string{}

	// TODO: Use parser.ParseDir?
	for _, src := range sources {
		src = path.Join(augments.Dir, src)
		f, err := a.natives.OpenFile(src)
		if err != nil {
			return fmt.Errorf("failed to open augmentation file %q: %s", src, err)
		}
		defer f.Close()
		parsed, err := parser.ParseFile(pkg.Fset, src, f, parser.ParseComments)
		if err != nil {
			return fmt.Errorf("failed to parse augmentation file %q: %s", src, err)
		}
		symbolFilter.Collect(parsed)
		extraAST = append(extraAST, parsed)
		extraFiles = append(extraFiles, src)
	}
	for _, f := range pkg.Syntax {
		symbolFilter.Prune(f)
	}
	pkg.Syntax = append(extraAST, pkg.Syntax...)
	pkg.CompiledGoFiles = append(extraFiles, pkg.CompiledGoFiles...)

	return nil
}
