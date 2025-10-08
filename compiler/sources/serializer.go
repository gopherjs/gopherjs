package sources

import (
	"encoding/gob"
	"go/ast"
	"go/token"
	"sync"
)

// Write will call encode multiple times to write the various fields
// of the sources. This is designed to be used with a gob.Encoder.
//
// The order of the calls must match the order of the calls in Read.
//
// s.TypeInfo, s.baseInfo, s.Package, and s.GoLinknames are intentionally
// omitted from encoding since they must be constructed in the context of the
// full program to be able to handle generics and cross-package references.
func (s *Sources) Write(encode func(any) error) error {
	prepareGob()
	if err := encode(s.ImportPath); err != nil {
		return err
	}
	if err := encode(s.Dir); err != nil {
		return err
	}
	files := make([]*ast.File, len(s.Files))
	for i, f := range s.Files {
		files[i] = prepareFile(f)
	}
	if err := encode(files); err != nil {
		return err
	}
	fs := s.FileSet
	if fs == nil {
		fs = token.NewFileSet()
	}
	if err := fs.Write(encode); err != nil {
		return err
	}
	if err := encode(s.JSFiles); err != nil {
		return err
	}
	return nil
}

// Read will call decode multiple times to read the various fields
// of the sources.
// The order of the calls must match the order of the calls in encodeSources.
//
// The given srcModTime is used to determine if the serialized sources are
// out-of-date. If they are then this returns a nil sources and nil error,
// and the build time that was read.
func (s *Sources) Read(decode func(any) error) error {
	prepareGob()
	if err := decode(&s.ImportPath); err != nil {
		return err
	}
	if err := decode(&s.Dir); err != nil {
		return err
	}
	if err := decode(&s.Files); err != nil {
		return err
	}
	for _, f := range s.Files {
		unpackFile(f)
	}
	if s.FileSet == nil {
		s.FileSet = token.NewFileSet()
	}
	if err := s.FileSet.Read(decode); err != nil {
		return err
	}
	return decode(&s.JSFiles)
}

// prepareFile is run when serializing a source to remove fields that can be
// easily reconstructed or are deprecated and should not be serialized.
// Returns a modified copy of the original file.
func prepareFile(file *ast.File) *ast.File {
	// Create a copy to avoid removing imports and comments from the original file.
	copy := *file
	file = &copy

	// Clear fields that can be easily reconstructed.
	file.Imports = nil
	file.Comments = nil

	// Clear fields that are deprecated.
	file.Scope = nil
	file.Unresolved = nil

	// Clear Obj fields to avoid serializing the deprecated data.
	// This will cause the original file to be modified but it's fine since Obj is deprecated.
	ast.Inspect(file, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok {
			id.Obj = nil
		}
		return true
	})
	return file
}

// unpackFile is run when deserializing a source to reconstruct the
// Imports and Comments fields that were cleared when serializing the file.
func unpackFile(file *ast.File) {
	var imports []*ast.ImportSpec
	var comments []*ast.CommentGroup
	ast.Inspect(file, func(n ast.Node) bool {
		if im, ok := n.(*ast.ImportSpec); ok {
			imports = append(imports, im)
		}
		if cg, ok := n.(*ast.CommentGroup); ok {
			comments = append(comments, cg)
		}
		return true
	})
	file.Imports = imports
	file.Comments = comments
}

// prepareGob registers the AST node types with the gob package
// so that the ast.File structs can be (de)serialized.
//
// This must be called before any (de)serialization is done.
// This can be called multiple times but the types will only be registered once.
//
// This only need to register the node types that can be referenced by
// an interface field in the AST.
var prepareGob = func() func() {
	registerTypes := func() {

		// Register expression nodes.
		gob.Register(&ast.BadExpr{})
		gob.Register(&ast.Ident{})
		gob.Register(&ast.Ellipsis{})
		gob.Register(&ast.BasicLit{})
		gob.Register(&ast.FuncLit{})
		gob.Register(&ast.CompositeLit{})
		gob.Register(&ast.ParenExpr{})
		gob.Register(&ast.SelectorExpr{})
		gob.Register(&ast.IndexExpr{})
		gob.Register(&ast.IndexListExpr{})
		gob.Register(&ast.SliceExpr{})
		gob.Register(&ast.TypeAssertExpr{})
		gob.Register(&ast.CallExpr{})
		gob.Register(&ast.StarExpr{})
		gob.Register(&ast.UnaryExpr{})
		gob.Register(&ast.BinaryExpr{})
		gob.Register(&ast.KeyValueExpr{})

		// Register type nodes.
		gob.Register(&ast.ArrayType{})
		gob.Register(&ast.StructType{})
		gob.Register(&ast.FuncType{})
		gob.Register(&ast.InterfaceType{})
		gob.Register(&ast.MapType{})
		gob.Register(&ast.ChanType{})

		// Register statement nodes.
		gob.Register(&ast.BadStmt{})
		gob.Register(&ast.DeclStmt{})
		gob.Register(&ast.EmptyStmt{})
		gob.Register(&ast.LabeledStmt{})
		gob.Register(&ast.ExprStmt{})
		gob.Register(&ast.SendStmt{})
		gob.Register(&ast.IncDecStmt{})
		gob.Register(&ast.AssignStmt{})
		gob.Register(&ast.GoStmt{})
		gob.Register(&ast.DeferStmt{})
		gob.Register(&ast.ReturnStmt{})
		gob.Register(&ast.BranchStmt{})
		gob.Register(&ast.BlockStmt{})
		gob.Register(&ast.IfStmt{})
		gob.Register(&ast.CaseClause{})
		gob.Register(&ast.SwitchStmt{})
		gob.Register(&ast.TypeSwitchStmt{})
		gob.Register(&ast.CommClause{})
		gob.Register(&ast.SelectStmt{})
		gob.Register(&ast.ForStmt{})
		gob.Register(&ast.RangeStmt{})

		// Register declaration nodes.
		gob.Register(&ast.BadDecl{})
		gob.Register(&ast.GenDecl{})
		gob.Register(&ast.FuncDecl{})

		// Register specification nodes.
		gob.Register(&ast.ImportSpec{})
		gob.Register(&ast.ValueSpec{})
		gob.Register(&ast.TypeSpec{})
	}

	var once sync.Once
	return func() {
		once.Do(registerTypes)
	}
}()
