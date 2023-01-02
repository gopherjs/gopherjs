package compiler

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"

	"github.com/neelance/astrewrite"
)

// sources is a slice of parsed Go sources.
//
// Note that the sources would normally belong to a single logical Go package,
// but they don't have to be a real Go package (i.e. found on the file system)
// or represent a complete package (i.e. it could be only a few source files
// compiled by `gopherjs build foo.go bar.go`).
type sources struct {
	// ImportPath representing the sources, if exists. May be empty for "virtual"
	// packages like testmain or playground-generated package.
	ImportPath string
	Files      []*ast.File
	FileSet    *token.FileSet
}

// Sort the Files slice by the original source name to ensure consistent order
// of processing. This is required for reproducible JavaScript output.
//
// Note this function mutates the original slice.
func (s sources) Sort() sources {
	sort.Slice(s.Files, func(i, j int) bool {
		return s.FileSet.File(s.Files[i].Pos()).Name() > s.FileSet.File(s.Files[j].Pos()).Name()
	})
	return s
}

// Simplify returns a new sources instance with each Files entry processed by
// astrewrite.Simplify.
func (s sources) Simplified(typesInfo *types.Info) sources {
	simplified := sources{
		ImportPath: s.ImportPath,
		FileSet:    s.FileSet,
	}
	for _, file := range s.Files {
		simplified.Files = append(simplified.Files, astrewrite.Simplify(file, typesInfo, false))
	}
	return simplified
}

// TypeCheck the sources. Returns information about declared package types and
// type information for the supplied AST.
func (s sources) TypeCheck(importContext *ImportContext) (*types.Info, *types.Package, error) {
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

	var typeErrs ErrorList

	importer := packageImporter{ImportContext: importContext}

	config := &types.Config{
		Importer: &importer,
		Sizes:    sizes32,
		Error:    func(err error) { typeErrs = typeErrs.AppendDistinct(err) },
	}
	typesPkg, err := config.Check(s.ImportPath, s.FileSet, s.Files, typesInfo)
	// If we encountered any import errors, it is likely that the other type errors
	// are not meaningful and would be resolved by fixing imports. Return them
	// separately, if any. https://github.com/gopherjs/gopherjs/issues/119.
	if importer.Errors.ErrOrNil() != nil {
		return nil, nil, importer.Errors.Trim(errLimit).ErrOrNil()
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
func (s sources) ParseGoLinknames() ([]GoLinkname, error) {
	goLinknames := []GoLinkname{}
	var errs ErrorList
	for _, file := range s.Files {
		found, err := parseGoLinknames(s.FileSet, s.ImportPath, file)
		errs = errs.Append(err)
		goLinknames = append(goLinknames, found...)
	}
	return goLinknames, errs.ErrOrNil()
}

// packageImporter implements go/types.Importer interface.
type packageImporter struct {
	ImportContext *ImportContext
	Errors        ErrorList
}

func (pi *packageImporter) Import(path string) (*types.Package, error) {
	if path == "unsafe" {
		return types.Unsafe, nil
	}

	a, err := pi.ImportContext.Import(path)
	if err != nil {
		pi.Errors = pi.Errors.AppendDistinct(err)
		return nil, err
	}

	return pi.ImportContext.Packages[a.ImportPath], nil
}
