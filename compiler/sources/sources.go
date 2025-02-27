package sources

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/jsFile"
	"github.com/gopherjs/gopherjs/compiler/linkname"
	"github.com/gopherjs/gopherjs/internal/errorList"
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
}

// Sort the Go files slice by the original source name to ensure consistent order
// of processing. This is required for reproducible JavaScript output.
//
// Note this function mutates the original Files slice.
func (s Sources) Sort() Sources {
	sort.Slice(s.Files, func(i, j int) bool {
		return s.FileSet.File(s.Files[i].Pos()).Name() > s.FileSet.File(s.Files[j].Pos()).Name()
	})
	return s
}

// Simplify processed each Files entry with astrewrite.Simplify.
//
// Note this function mutates the original Files slice.
func (s Sources) Simplify(typesInfo *types.Info) {
	for i, file := range s.Files {
		s.Files[i] = astrewrite.Simplify(file, typesInfo, false)
	}
}

// TypeCheck the sources. Returns information about declared package types and
// type information for the supplied AST.
func (s Sources) TypeCheck(importer types.Importer, sizes types.Sizes, tContext *types.Context) (*types.Info, *types.Package, error) {
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
		return nil, nil, ecImporter.Errors.Trim(errLimit).ErrOrNil()
	}
	// Return any other type errors.
	if typeErrs.ErrOrNil() != nil {
		return nil, nil, typeErrs.Trim(errLimit).ErrOrNil()
	}
	// Any general errors that may have occurred during type checking.
	if err != nil {
		return nil, nil, err
	}
	return typesInfo, typesPkg, nil
}

// ParseGoLinknames extracts all //go:linkname compiler directive from the sources.
func (s Sources) ParseGoLinknames() ([]linkname.GoLinkname, error) {
	goLinknames := []linkname.GoLinkname{}
	var errs errorList.ErrorList
	for _, file := range s.Files {
		found, err := linkname.ParseGoLinknames(s.FileSet, s.ImportPath, file)
		errs = errs.Append(err)
		goLinknames = append(goLinknames, found...)
	}
	return goLinknames, errs.ErrOrNil()
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
