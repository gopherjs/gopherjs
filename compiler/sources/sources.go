package sources

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"github.com/neelance/astrewrite"

	"github.com/gopherjs/gopherjs/compiler/internal/analysis"
	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
	"github.com/gopherjs/gopherjs/compiler/jsFile"
	"github.com/gopherjs/gopherjs/compiler/linkname"
	"github.com/gopherjs/gopherjs/internal/errorList"
	"github.com/gopherjs/gopherjs/internal/experiments"
)

// Sources is a slice of parsed Go sources and additional data for a package.
//
// Note that the sources would normally belong to a single logical Go package,
// but they don't have to be a real Go package (i.e. found on the file system)
// or represent a complete package (i.e. it could be only a few source files
// compiled by `gopherjs build foo.go bar.go`).
type Sources struct {
	// ImportPath representing the sources, if exists.
	//
	// May be empty for "virtual"
	// packages like testmain or playground-generated package.
	// Otherwise this must be the absolute import path for a package.
	ImportPath string

	// Dir is the directory containing package sources
	Dir string

	// Files is the parsed and augmented Go AST files for the package.
	Files []*ast.File

	// FileSet is the file set for the parsed files.
	FileSet *token.FileSet

	// JSFiles is the JavaScript files that are part of the package.
	JSFiles []jsFile.JSFile

	// TypeInfo is the type information this package.
	// This is nil until set by Analyze.
	TypeInfo *analysis.Info

	// baseInfo is the base type information this package.
	// This is nil until set by TypeCheck.
	baseInfo *types.Info

	// Package is the types package for these source files.
	// This is nil until set by TypeCheck.
	Package *types.Package

	// GoLinknames is the set of Go linknames for this package.
	// This is nil until set by ParseGoLinknames.
	GoLinknames []linkname.GoLinkname
}

type Importer func(path, srcDir string) (*Sources, error)

// sort the Go files slice by the original source name to ensure consistent order
// of processing. This is required for reproducible JavaScript output.
//
// Note this function mutates the original Files slice.
func (s *Sources) Sort() {
	sort.Slice(s.Files, func(i, j int) bool {
		return s.getFileName(s.Files[i]) > s.getFileName(s.Files[j])
	})
}

func (s *Sources) getFileName(file *ast.File) string {
	return s.FileSet.File(file.Pos()).Name()
}

// Simplify processed each Files entry with astrewrite.Simplify.
//
// Note this function mutates the original Files slice.
// This must be called after TypeCheck and before analyze since
// this will change the pointers in the AST. For example, the pointers
// to function literals will change, making it impossible to find them
// in the type information, if analyze is called first.
func (s *Sources) Simplify() {
	for i, file := range s.Files {
		s.Files[i] = astrewrite.Simplify(file, s.baseInfo, false)
	}
}

// TypeCheck the sources. Returns information about declared package types and
// type information for the supplied AST.
// This will set the Package field on the Sources.
//
// If the Package field is not nil, e.g. this function has already been run,
// this will be a no-op.
//
// This must be called prior to simplify to get the types.Info used by simplify.
func (s *Sources) TypeCheck(importer Importer, sizes types.Sizes, tContext *types.Context) error {
	if s.Package != nil && s.baseInfo != nil {
		// type checking has already been done so return early.
		return nil
	}

	const errLimit = 10 // Max number of type checking errors to return.

	typesInfo := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Implicits:  make(map[ast.Node]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
		Scopes:     make(map[ast.Node]*types.Scope),
		Instances:  make(map[*ast.Ident]types.Instance),
	}

	var typeErrs errorList.ErrorList

	pkgImporter := &packageImporter{
		srcDir:   s.Dir,
		importer: importer,
		sizes:    sizes,
		tContext: tContext,
	}

	config := &types.Config{
		Context:  tContext,
		Importer: pkgImporter,
		Sizes:    sizes,
		Error:    func(err error) { typeErrs = typeErrs.AppendDistinct(err) },
	}
	typesPkg, err := config.Check(s.ImportPath, s.FileSet, s.Files, typesInfo)
	// If we encountered any import errors, it is likely that the other type errors
	// are not meaningful and would be resolved by fixing imports. Return them
	// separately, if any. https://github.com/gopherjs/gopherjs/issues/119.
	if pkgImporter.Errors.ErrOrNil() != nil {
		return pkgImporter.Errors.Trim(errLimit).ErrOrNil()
	}
	// Return any other type errors.
	if typeErrs.ErrOrNil() != nil {
		return typeErrs.Trim(errLimit).ErrOrNil()
	}
	// Any general errors that may have occurred during type checking.
	if err != nil {
		return err
	}

	// If generics are not enabled, ensure the package does not requires generics support.
	if !experiments.Env.Generics {
		if genErr := typeparams.RequiresGenericsSupport(typesInfo); genErr != nil {
			return fmt.Errorf("some packages requires generics support (https://github.com/gopherjs/gopherjs/issues/1013): %w", genErr)
		}
	}

	s.baseInfo = typesInfo
	s.Package = typesPkg
	return nil
}

// CollectInstances will determine the type parameters instances for the package.
//
// This must be called before Analyze to have the type parameters instances
// needed during analysis.
func (s *Sources) CollectInstances(tContext *types.Context, instances *typeparams.PackageInstanceSets) {
	tc := typeparams.Collector{
		TContext:  tContext,
		Info:      s.baseInfo,
		Instances: instances,
	}
	tc.Scan(s.Package, s.Files...)
}

// Analyze will determine the type parameters instances, blocking,
// and other type information for the package.
// This will set the TypeInfo and Instances fields on the Sources.
//
// This must be called after to simplify to ensure the pointers
// in the AST are still valid.
// The instances must be collected prior to this call.
//
// Note that at the end of this call the analysis information
// has NOT been propagated across packages yet.
func (s *Sources) Analyze(importer Importer, tContext *types.Context, instances *typeparams.PackageInstanceSets) {
	infoImporter := func(path string) (*analysis.Info, error) {
		srcs, err := importer(path, s.Dir)
		if err != nil {
			return nil, err
		}
		return srcs.TypeInfo, nil
	}
	s.TypeInfo = analysis.AnalyzePkg(s.Files, s.FileSet, s.baseInfo, tContext, s.Package, instances, infoImporter)
}

// ParseGoLinknames extracts all //go:linkname compiler directive from the sources.
//
// This will set the GoLinknames field on the Sources.
func (s *Sources) ParseGoLinknames() error {
	goLinknames := []linkname.GoLinkname{}
	var errs errorList.ErrorList
	for _, file := range s.Files {
		found, err := linkname.ParseGoLinknames(s.FileSet, s.ImportPath, file)
		errs = errs.Append(err)
		goLinknames = append(goLinknames, found...)
	}
	if err := errs.ErrOrNil(); err != nil {
		return err
	}
	s.GoLinknames = goLinknames
	return nil
}

// UnresolvedImports calculates the import paths of the package's dependencies
// based on all the imports in the augmented Go AST files.
//
// This is used to determine the unresolved imports that weren't in the
// PackageData.Imports slice since they were added during augmentation or
// during template generation.
//
// The given skip paths (typically those imports from PackageData.Imports)
// will not be returned in the results.
// This will not return any `*_test` packages in the results.
func (s *Sources) UnresolvedImports(skip ...string) []string {
	seen := make(map[string]struct{})
	for _, sk := range skip {
		seen[sk] = struct{}{}
	}
	imports := []string{}
	for _, file := range s.Files {
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if _, ok := seen[path]; !ok {
				if !strings.HasSuffix(path, "_test") {
					imports = append(imports, path)
				}
				seen[path] = struct{}{}
			}
		}
	}
	sort.Strings(imports)
	return imports
}

// packageImporter implements go/types.Importer interface and
// wraps it to collect import errors.
type packageImporter struct {
	srcDir   string
	importer Importer
	sizes    types.Sizes
	tContext *types.Context
	Errors   errorList.ErrorList
}

func (pi *packageImporter) Import(path string) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}

	srcs, err := pi.importer(path, pi.srcDir)
	if err != nil {
		pi.Errors = pi.Errors.AppendDistinct(err)
		return nil, err
	}

	// If the sources doesn't have the package determined yet, get it now,
	// otherwise this will be a no-op.
	// This will recursively get the packages for all of it's dependencies too.
	err = srcs.TypeCheck(pi.importer, pi.sizes, pi.tContext)
	if err != nil {
		pi.Errors = pi.Errors.AppendDistinct(err)
		return nil, err
	}

	return srcs.Package, nil
}

// SortedSourcesSlice in place sorts the given slice of Sources by ImportPath.
// This will not change the order of the files within any Sources.
func SortedSourcesSlice(sourcesSlice []*Sources) {
	sort.Slice(sourcesSlice, func(i, j int) bool {
		return sourcesSlice[i].ImportPath < sourcesSlice[j].ImportPath
	})
}
