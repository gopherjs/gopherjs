package sources

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/internal/analysis"
	"github.com/gopherjs/gopherjs/compiler/internal/typeparams"
	"github.com/gopherjs/gopherjs/compiler/jsFile"
	"github.com/gopherjs/gopherjs/compiler/linkname"
	"github.com/gopherjs/gopherjs/internal/errorList"
	"github.com/gopherjs/gopherjs/internal/experiments"
	"github.com/neelance/astrewrite"
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
	// This is nil until PrepareInfo is called.
	TypeInfo *analysis.Info

	// Instances is the type parameters instances for this package.
	// This is nil until PrepareInfo is called.
	Instances *typeparams.PackageInstanceSets

	// Package is the type-PrepareInfo package.
	// This is nil until PrepareInfo is called.
	Package *types.Package

	// GoLinknames is the set of Go linknames for this package.
	// This is nil until PrepareInfo is called.
	GoLinknames []linkname.GoLinkname
}

type SourcesImporter func(path, srcDir string) (*Sources, error)

// Prepare recursively processes the provided sources and
// prepares them for compilation by sorting the files by name,
// determining the type information, go linknames, and simplifying the AST.
//
// The importer function is used to import the sources of other packages
// that are imported by the package being prepared. The other sources must
// be prepared prior to being returned by the importer so that the type
// information can be used.
//
// Note that at the end of this call the analysis information
// has NOT been propagated across packages yet.
func (s *Sources) Prepare(importer SourcesImporter, sizes types.Sizes, tContext *types.Context) error {
	s.sort()

	typesImporter := func(path string) (*types.Package, error) {
		if path == "unsafe" {
			return types.Unsafe, nil
		}

	}

	typesInfo, err := s.typeCheck(typesImporter, sizes, tContext)
	if err != nil {
		return err
	}

	if !experiments.Env.Generics {
		if genErr := typeparams.RequiresGenericsSupport(typesInfo); genErr != nil {
			return fmt.Errorf("package %s requires generics support (https://github.com/gopherjs/gopherjs/issues/1013): %w", s.ImportPath, genErr)
		}
	}

	s.analyze(typesInfo, importer, tContext)

	// Extract all go:linkname compiler directives from the package source.
	if err := s.parseGoLinknames(); err != nil {
		return err
	}

	s.simplify()
	return nil
}

// sort the Go files slice by the original source name to ensure consistent order
// of processing. This is required for reproducible JavaScript output.
//
// Note this function mutates the original Files slice.
func (s Sources) sort() {
	sort.Slice(s.Files, func(i, j int) bool {
		return s.FileSet.File(s.Files[i].Pos()).Name() > s.FileSet.File(s.Files[j].Pos()).Name()
	})
}

// simplify processed each Files entry with astrewrite.Simplify.
//
// Note this function mutates the original Files slice.
// This must be called after TypeCheck.
func (s Sources) simplify() {
	for i, file := range s.Files {
		s.Files[i] = astrewrite.Simplify(file, s.TypeInfo.Info, false)
	}
}

// typeCheck the sources. Returns information about declared package types and
// type information for the supplied AST.
//
// This must be called prior to Simplify.
func (s Sources) typeCheck(importer types.Importer, sizes types.Sizes, tContext *types.Context) (*types.Info, error) {
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

	ecImporter := &errorCollectingImporter{Importer: importer}

	config := &types.Config{
		Context:  tContext,
		Importer: ecImporter,
		Sizes:    sizes,
		Error:    func(err error) { typeErrs = typeErrs.AppendDistinct(err) },
	}
	typesPkg, err := config.Check(s.ImportPath, s.FileSet, s.Files, typesInfo)
	// If we encountered any import errors, it is likely that the other type errors
	// are not meaningful and would be resolved by fixing imports. Return them
	// separately, if any. https://github.com/gopherjs/gopherjs/issues/119.
	if ecImporter.Errors.ErrOrNil() != nil {
		return nil, ecImporter.Errors.Trim(errLimit).ErrOrNil()
	}
	// Return any other type errors.
	if typeErrs.ErrOrNil() != nil {
		return nil, typeErrs.Trim(errLimit).ErrOrNil()
	}
	// Any general errors that may have occurred during type checking.
	if err != nil {
		return nil, err
	}

	s.Package = typesPkg
	return typesInfo, nil
}

// analyze will determine the type parameters instances, blocking,
// and other type information for the package.
//
// This must be called prior to Simplify.
// Note that at the end of this call the analysis information
// has NOT been propagated across packages yet.
func (s Sources) analyze(typesInfo *types.Info, importer SourcesImporter, tContext *types.Context) {
	tc := typeparams.Collector{
		TContext:  tContext,
		Info:      typesInfo,
		Instances: &typeparams.PackageInstanceSets{},
	}
	tc.Scan(s.Package, s.Files...)

	infoImporter := func(path string) (*analysis.Info, error) {
		srcs, err := importer(path)
		if err != nil {
			return nil, err
		}
		return srcs.TypeInfo, nil
	}
	anaInfo := analysis.AnalyzePkg(s.Files, s.FileSet, typesInfo, tContext, s.Package, tc.Instances, infoImporter)

	s.TypeInfo = anaInfo
	s.Instances = tc.Instances
}

// parseGoLinknames extracts all //go:linkname compiler directive from the sources.
//
// This will set the GoLinknames field on the Sources struct.
// This must be called prior to Simplify.
func (s Sources) parseGoLinknames() error {
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
func (s Sources) UnresolvedImports(skip ...string) []string {
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
