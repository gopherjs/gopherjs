package build

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path"
	"strconv"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/astutil"
)

// overrideInfo is used by parseAndAugment methods to manage
// directives and how the overlay and original are merged.
type overrideInfo struct {
	// KeepOriginal indicates that the original code should be kept
	// but the identifier will be prefixed by `_gopherjs_original_foo`.
	// If false the original code is removed.
	keepOriginal bool

	// purgeMethods indicates that this info is for a type and
	// if a method has this type as a receiver should also be removed.
	// If the method is defined in the overlays and therefore has its
	// own overrides, this will be ignored.
	purgeMethods bool

	// overrideSignature is the function definition given in the overlays
	// that should be used to replace the signature in the originals.
	// Only receivers, type parameters, parameters, and results will be used.
	overrideSignature *ast.FuncDecl
}

// pkgOverrideInfo is the collection of overrides still needed for a package.
type pkgOverrideInfo struct {
	// overrides is a map of identifier to overrideInfo to override
	// individual named structs, interfaces, functions, and methods.
	overrides map[string]overrideInfo

	// overlayFiles are the files from the natives that still haven't been
	// appended to a file from the package, typically the first file.
	overlayFiles []*ast.File

	// jsFiles are the additional JS files that are part of the natives.
	jsFiles []JSFile
}

// Augmentor is an on-the-fly package augmentor.
//
// When a file from a package is being parsed, the Augmentor will augment
// the AST with the changes loaded from the native overrides.
// The augmentor will hold onto the override information for additional files
// that come from the same package. This is designed to be used with
// `x/tools/go/packages.Load` as a middleware in the parse file step via
// `Config.ParseFile`.
//
// The first file from a package will have any additional methods and
// information from the natives injected into the AST. All files from a package
// will be augmented by the overrides.
type Augmentor struct {
	// packages is a map of package import path to the package's override.
	// This is used to keep track of the overrides for a package and indicate
	// that additional files from the natives have already been applied.
	packages map[string]*pkgOverrideInfo
}

func (aug *Augmentor) Augment(xctx XContext, pkg *PackageData, fileSet *token.FileSet, file *ast.File) error {
	pkgAug := aug.getPackageOverrides(xctx, pkg, fileSet)

	augmentOriginalImports(pkg.ImportPath, file)

	if len(pkgAug.overrides) > 0 {
		augmentOriginalFile(file, pkgAug.overrides)
	}

	if len(pkgAug.overlayFiles) > 0 {
		// Append the overlay files to the first file of the package.
		// This is to ensure that the package is augmented with all the
		// additional methods and information from the natives.
		err := astutil.ConcatenateFiles(file, pkgAug.overlayFiles...)
		if err != nil {
			return fmt.Errorf("failed to concatenate overlay files onto %q: %w", fileSet.Position(file.Package).Filename, err)
		}
		pkgAug.overlayFiles = nil

		// TODO: REMOVE
		if file.Name.Name == "sync" {
			buf := &bytes.Buffer{}
			if err := format.Node(buf, fileSet, file); err != nil {
				panic(fmt.Errorf("failed to format augmented file: %w", err))
			}
			fmt.Println(">>>>>\n", buf.String(), "\n<<<<<")
			fmt.Println(">>>>>")
			ast.Print(fileSet, file)
			fmt.Println("\n<<<<<")
		}
	}

	return nil
}

func (aug *Augmentor) GetJSFiles(pkg *PackageData) []JSFile {
	pkgAug, ok := aug.packages[pkg.ImportPath]
	if !ok {
		return nil
	}
	return pkgAug.jsFiles
}

// getPackageOverrides looks up an already loaded package override
// or loads the package's natives, parses the overlay files, and
// stores the overrides for the package in the augmentor for next time.
func (aug *Augmentor) getPackageOverrides(xctx XContext, pkg *PackageData, fileSet *token.FileSet) *pkgOverrideInfo {
	importPath := pkg.ImportPath
	if pkgAug, ok := aug.packages[importPath]; ok {
		return pkgAug
	}

	jsFiles, overlayFiles := parseOverlayFiles(xctx, pkg, fileSet)

	overrides := make(map[string]overrideInfo)
	for _, file := range overlayFiles {
		augmentOverlayFile(file, overrides)
	}
	delete(overrides, `init`)

	pkgAug := &pkgOverrideInfo{
		overrides:    overrides,
		overlayFiles: overlayFiles,
		jsFiles:      jsFiles,
	}

	if aug.packages == nil {
		aug.packages = map[string]*pkgOverrideInfo{}
	}
	aug.packages[importPath] = pkgAug
	return pkgAug
}

// parseOverlayFiles loads and parses overlay files
// to augment the original files with.
func parseOverlayFiles(xctx XContext, pkg *PackageData, fileSet *token.FileSet) ([]JSFile, []*ast.File) {
	importPath := pkg.ImportPath
	isXTest := strings.HasSuffix(importPath, "_test")
	if isXTest {
		importPath = importPath[:len(importPath)-5]
	}

	nativesContext := overlayCtx(xctx.Env())
	nativesPkg, err := nativesContext.Import(importPath, "", 0)
	if err != nil {
		return nil, nil
	}

	jsFiles := nativesPkg.JSFiles
	var files []*ast.File
	names := nativesPkg.GoFiles
	if pkg.IsTest {
		names = append(names, nativesPkg.TestGoFiles...)
	}
	if isXTest {
		names = nativesPkg.XTestGoFiles
	}

	for _, name := range names {
		fullPath := path.Join(nativesPkg.Dir, name)
		r, err := nativesContext.bctx.OpenFile(fullPath)
		if err != nil {
			panic(err)
		}
		// Files should be uniquely named and in the original package directory in order to be
		// ordered correctly
		newPath := path.Join(pkg.Dir, "gopherjs__"+name)
		file, err := parser.ParseFile(fileSet, newPath, r, parser.ParseComments)
		if err != nil {
			panic(err)
		}
		r.Close()

		files = append(files, file)
	}
	return jsFiles, files
}

// augmentOverlayFile is the part of parseAndAugment that processes
// an overlay file AST to collect information such as compiler directives
// and perform any initial augmentation needed to the overlay.
func augmentOverlayFile(file *ast.File, overrides map[string]overrideInfo) {
	anyChange := false
	for i, decl := range file.Decls {
		purgeDecl := astutil.Purge(decl)
		switch d := decl.(type) {
		case *ast.FuncDecl:
			k := astutil.FuncKey(d)
			oi := overrideInfo{
				keepOriginal: astutil.KeepOriginal(d),
			}
			if astutil.OverrideSignature(d) {
				oi.overrideSignature = d
				purgeDecl = true
			}
			overrides[k] = oi
		case *ast.GenDecl:
			for j, spec := range d.Specs {
				purgeSpec := purgeDecl || astutil.Purge(spec)
				switch s := spec.(type) {
				case *ast.TypeSpec:
					overrides[s.Name.Name] = overrideInfo{
						purgeMethods: purgeSpec,
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						overrides[name.Name] = overrideInfo{}
					}
				}
				if purgeSpec {
					anyChange = true
					d.Specs[j] = nil
				}
			}
		}
		if purgeDecl {
			anyChange = true
			file.Decls[i] = nil
		}
	}
	if anyChange {
		astutil.FinalizeRemovals(file)
		astutil.PruneImports(file)
	}
}

// augmentOriginalImports is the part of parseAndAugment that processes
// an original file AST to modify the imports for that file.
func augmentOriginalImports(importPath string, file *ast.File) {
	switch importPath {
	case "crypto/rand", "encoding/gob", "encoding/json", "expvar", "go/token", "log", "math/big", "math/rand", "regexp", "time":
		for _, spec := range file.Imports {
			path, _ := strconv.Unquote(spec.Path.Value)
			if path == "sync" {
				if spec.Name == nil {
					spec.Name = ast.NewIdent("sync")
				}
				spec.Path.Value = `"github.com/gopherjs/gopherjs/nosync"`
			}
		}
	}
}

// augmentOriginalFile is the part of parseAndAugment that processes an
// original file AST to augment the source code using the overrides from
// the overlay files.
func augmentOriginalFile(file *ast.File, overrides map[string]overrideInfo) {
	anyChange := false
	for i, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if info, ok := overrides[astutil.FuncKey(d)]; ok {
				anyChange = true
				removeFunc := true
				if info.keepOriginal {
					// Allow overridden function calls
					// The standard library implementation of foo() becomes _gopherjs_original_foo()
					d.Name.Name = "_gopherjs_original_" + d.Name.Name
					removeFunc = false
				}
				if overSig := info.overrideSignature; overSig != nil {
					d.Recv = overSig.Recv
					d.Type.TypeParams = overSig.Type.TypeParams
					d.Type.Params = overSig.Type.Params
					d.Type.Results = overSig.Type.Results
					removeFunc = false
				}
				if removeFunc {
					file.Decls[i] = nil
				}
			} else if recvKey := astutil.FuncReceiverKey(d); len(recvKey) > 0 {
				// check if the receiver has been purged, if so, remove the method too.
				if info, ok := overrides[recvKey]; ok && info.purgeMethods {
					anyChange = true
					file.Decls[i] = nil
				}
			}
		case *ast.GenDecl:
			for j, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if _, ok := overrides[s.Name.Name]; ok {
						anyChange = true
						d.Specs[j] = nil
					}
				case *ast.ValueSpec:
					if len(s.Names) == len(s.Values) {
						// multi-value context
						// e.g. var a, b = 2, foo[int]()
						// A removal will also remove the value which may be from a
						// function call. This allows us to remove unwanted statements.
						// However, if that call has a side effect which still needs
						// to be run, add the call into the overlay.
						for k, name := range s.Names {
							if _, ok := overrides[name.Name]; ok {
								anyChange = true
								s.Names[k] = nil
								s.Values[k] = nil
							}
						}
					} else {
						// single-value context
						// e.g. var a, b = foo[int]()
						// If a removal from the overlays makes all returned values unused,
						// then remove the function call as well. This allows us to stop
						// unwanted calls if needed. If that call has a side effect which
						// still needs to be run, add the call into the overlay.
						nameRemoved := false
						for _, name := range s.Names {
							if _, ok := overrides[name.Name]; ok {
								nameRemoved = true
								name.Name = `_`
							}
						}
						if nameRemoved {
							removeSpec := true
							for _, name := range s.Names {
								if name.Name != `_` {
									removeSpec = false
									break
								}
							}
							if removeSpec {
								anyChange = true
								d.Specs[j] = nil
							}
						}
					}
				}
			}
		}
	}
	if anyChange {
		astutil.FinalizeRemovals(file)
		astutil.PruneImports(file)
	}
}
