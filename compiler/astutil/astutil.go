package astutil

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func RemoveParens(e ast.Expr) ast.Expr {
	for {
		p, isParen := e.(*ast.ParenExpr)
		if !isParen {
			return e
		}
		e = p.X
	}
}

func SetType(info *types.Info, t types.Type, e ast.Expr) ast.Expr {
	info.Types[e] = types.TypeAndValue{Type: t}
	return e
}

func NewIdent(name string, t types.Type, info *types.Info, pkg *types.Package) *ast.Ident {
	ident := ast.NewIdent(name)
	info.Types[ident] = types.TypeAndValue{Type: t}
	obj := types.NewVar(0, pkg, name, t)
	info.Uses[ident] = obj
	return ident
}

// IsTypeExpr returns true if expr denotes a type. This can be used to
// distinguish between calls and type conversions.
func IsTypeExpr(expr ast.Expr, info *types.Info) bool {
	// Note that we could've used info.Types[expr].IsType() instead of doing our
	// own analysis. However, that creates a problem because we synthesize some
	// *ast.CallExpr nodes and, more importantly, *ast.Ident nodes that denote a
	// type. Unfortunately, because the flag that controls
	// types.TypeAndValue.IsType() return value is unexported we wouldn't be able
	// to set it correctly. Thus, we can't rely on IsType().
	switch e := expr.(type) {
	case *ast.ArrayType, *ast.ChanType, *ast.FuncType, *ast.InterfaceType, *ast.MapType, *ast.StructType:
		return true
	case *ast.StarExpr:
		return IsTypeExpr(e.X, info)
	case *ast.Ident:
		_, ok := info.Uses[e].(*types.TypeName)
		return ok
	case *ast.SelectorExpr:
		_, ok := info.Uses[e.Sel].(*types.TypeName)
		return ok
	case *ast.IndexExpr:
		ident, ok := e.X.(*ast.Ident)
		if !ok {
			return false
		}
		_, ok = info.Uses[ident].(*types.TypeName)
		return ok
	case *ast.IndexListExpr:
		ident, ok := e.X.(*ast.Ident)
		if !ok {
			return false
		}
		_, ok = info.Uses[ident].(*types.TypeName)
		return ok
	case *ast.ParenExpr:
		return IsTypeExpr(e.X, info)
	default:
		return false
	}
}

func ImportsUnsafe(file *ast.File) bool {
	for _, imp := range file.Imports {
		if imp.Path.Value == `"unsafe"` {
			return true
		}
	}
	return false
}

// ImportName tries to determine the package name for an import.
//
// If the package name isn't specified then this will make a best
// make a best guess using the import path.
// If the import name is dot (`.`), blank (`_`), or there
// was an issue determining the package name then empty is returned.
func ImportName(spec *ast.ImportSpec) string {
	var name string
	if spec.Name != nil {
		name = spec.Name.Name
	} else {
		importPath, _ := strconv.Unquote(spec.Path.Value)
		name = path.Base(importPath)
	}

	switch name {
	case `_`, `.`, `/`:
		return ``
	default:
		return name
	}
}

// FuncKey returns a string, which uniquely identifies a top-level function or
// method in a package.
func FuncKey(d *ast.FuncDecl) string {
	if recvKey := FuncReceiverKey(d); len(recvKey) > 0 {
		return recvKey + "." + d.Name.Name
	}
	return d.Name.Name
}

// FuncReceiverKey returns a string that uniquely identifies the receiver
// struct of the function or an empty string if there is no receiver.
// This name will match the name of the struct in the struct's type spec.
func FuncReceiverKey(d *ast.FuncDecl) string {
	if d == nil || d.Recv == nil || len(d.Recv.List) == 0 {
		return ``
	}
	recv := d.Recv.List[0].Type
	for {
		switch r := recv.(type) {
		case *ast.IndexListExpr:
			recv = r.X
			continue
		case *ast.IndexExpr:
			recv = r.X
			continue
		case *ast.StarExpr:
			recv = r.X
			continue
		case *ast.Ident:
			return r.Name
		default:
			panic(fmt.Errorf(`unexpected type %T in receiver of function: %v`, recv, d))
		}
	}
}

// KeepOriginal returns true if gopherjs:keep-original directive is present
// before a function decl.
//
// `//gopherjs:keep-original` is a GopherJS-specific directive, which can be
// applied to functions in native overlays and will instruct the augmentation
// logic to expose the original function such that it can be called. For a
// function in the original called `foo`, it will be accessible by the name
// `_gopherjs_original_foo`.
func KeepOriginal(d *ast.FuncDecl) bool {
	return hasDirective(d, `keep-original`)
}

// Purge returns true if gopherjs:purge directive is present
// on a struct, interface, type, variable, constant, or function.
//
// `//gopherjs:purge` is a GopherJS-specific directive, which can be
// applied in native overlays and will instruct the augmentation logic to
// delete part of the standard library without a replacement. This directive
// can be used to remove code that would be invalid in GopherJS, such as code
// using unsupported features (e.g. generic interfaces before generics were
// fully supported). It should be used with caution since it may remove needed
// dependencies. If a type is purged, all methods using that type as
// a receiver will also be purged.
func Purge(d ast.Node) bool {
	return hasDirective(d, `purge`)
}

// OverrideSignature returns true if gopherjs:override-signature directive is
// present on a function.
//
// `//gopherjs:override-signature` is a GopherJS-specific directive, which can
// be applied in native overlays and will instruct the augmentation logic to
// replace the original function signature which has the same FuncKey with the
// signature defined in the native overlays.
// This directive can be used to remove generics from a function signature or
// to replace a receiver of a function with another one. The given native
// overlay function will be removed, so no method body is needed in the overlay.
//
// The new signature may not contain types which require a new import since
// the imports will not be automatically added when needed, only removed.
// Use a type alias in the overlay to deal manage imports.
func OverrideSignature(d *ast.FuncDecl) bool {
	return hasDirective(d, `override-signature`)
}

// directiveMatcher is a regex which matches a GopherJS directive
// and finds the directive action.
var directiveMatcher = regexp.MustCompile(`^\/(?:\/|\*)gopherjs:([\w-]+)`)

// hasDirective returns true if the associated documentation
// or line comments for the given node have the given directive action.
//
// All GopherJS-specific directives must start with `//gopherjs:` or
// `/*gopherjs:` and followed by an action without any whitespace. The action
// must be one or more letter, decimal, underscore, or hyphen.
//
// see https://pkg.go.dev/cmd/compile#hdr-Compiler_Directives
func hasDirective(node ast.Node, directiveAction string) bool {
	foundDirective := false
	ast.Inspect(node, func(n ast.Node) bool {
		switch a := n.(type) {
		case *ast.Comment:
			m := directiveMatcher.FindStringSubmatch(a.Text)
			if len(m) == 2 && m[1] == directiveAction {
				foundDirective = true
			}
			return false
		case *ast.CommentGroup:
			return !foundDirective
		default:
			return n == node
		}
	})
	return foundDirective
}

// HasDirectivePrefix determines if any line in the given file
// has the given directive prefix in it.
func HasDirectivePrefix(file *ast.File, prefix string) bool {
	foundDirective := false
	ast.Inspect(file, func(n ast.Node) bool {
		if c, ok := n.(*ast.Comment); ok && strings.HasPrefix(c.Text, prefix) {
			foundDirective = true
		}
		return !foundDirective
	})
	return foundDirective
}

// FindLoopStmt tries to find the loop statement among the AST nodes in the
// |stack| that corresponds to the break/continue statement represented by
// branch.
//
// This function is label-aware and assumes the code was successfully
// type-checked.
func FindLoopStmt(stack []ast.Node, branch *ast.BranchStmt, typeInfo *types.Info) ast.Stmt {
	if branch.Tok != token.CONTINUE && branch.Tok != token.BREAK {
		panic(fmt.Errorf("FindLoopStmt() must be used with a break or continue statement only, got: %v", branch))
	}

	for i := len(stack) - 1; i >= 0; i-- {
		n := stack[i]

		if branch.Label != nil {
			// For a labelled continue the loop will always be in a labelled statement.
			referencedLabel := typeInfo.Uses[branch.Label].(*types.Label)
			labelStmt, ok := n.(*ast.LabeledStmt)
			if !ok {
				continue
			}
			if definedLabel := typeInfo.Defs[labelStmt.Label]; definedLabel != referencedLabel {
				continue
			}
			n = labelStmt.Stmt
		}

		switch s := n.(type) {
		case *ast.RangeStmt, *ast.ForStmt:
			return s.(ast.Stmt)
		}
	}

	// This should never happen in a source that passed type checking.
	panic(fmt.Errorf("continue/break statement %v doesn't have a matching loop statement among ancestors", branch))
}

// EndsWithReturn returns true if the last effective statement is a "return".
func EndsWithReturn(stmts []ast.Stmt) bool {
	if len(stmts) == 0 {
		return false
	}
	last := stmts[len(stmts)-1]
	switch l := last.(type) {
	case *ast.ReturnStmt:
		return true
	case *ast.LabeledStmt:
		return EndsWithReturn([]ast.Stmt{l.Stmt})
	case *ast.BlockStmt:
		return EndsWithReturn(l.List)
	default:
		return false
	}
}

// isOnlyImports determines if this file is empty except for imports.
func isOnlyImports(file *ast.File) bool {
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			continue
		}

		// The decl was either a FuncDecl or a non-import GenDecl.
		return false
	}
	return true
}

// PruneImports will remove any unused imports from the file.
//
// This will not remove any dot (`.`) or blank (`_`) imports, unless
// there are no declarations or directives meaning that all the imports
// should be cleared.
// If the removal of code causes an import to be removed, the init's from that
// import may not be run anymore. If we still need to run an init for an import
// which is no longer used, add it to the overlay as a blank (`_`) import.
//
// This uses the given name or guesses at the name using the import path,
// meaning this doesn't work for packages which have a different package name
// from the path, including those paths which are versioned
// (e.g. `github.com/foo/bar/v2` where the package name is `bar`)
// or if the import is defined using a relative path (e.g. `./..`).
// Those cases don't exist in the native for Go, so we should only run
// this pruning when we have native overlays, but not for unknown packages.
func PruneImports(file *ast.File) {
	if isOnlyImports(file) && !HasDirectivePrefix(file, `//go:linkname `) {
		// The file is empty, remove all imports including any `.` or `_` imports.
		file.Imports = nil
		file.Decls = nil
		return
	}

	unused := make(map[string]int, len(file.Imports))
	for i, in := range file.Imports {
		if name := ImportName(in); len(name) > 0 {
			unused[name] = i
		}
	}

	// Remove from "unused imports" for any import which is used.
	ast.Inspect(file, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Obj == nil {
				delete(unused, id.Name)
			}
		}
		return len(unused) > 0
	})
	if len(unused) == 0 {
		return
	}

	// Remove from "unused imports" for any import used for a directive.
	directiveImports := map[string]string{
		`unsafe`: `//go:linkname `,
		`embed`:  `//go:embed `,
	}
	for name, index := range unused {
		in := file.Imports[index]
		path, _ := strconv.Unquote(in.Path.Value)
		directivePrefix, hasPath := directiveImports[path]
		if hasPath && HasDirectivePrefix(file, directivePrefix) {
			// since the import is otherwise unused set the name to blank.
			in.Name = ast.NewIdent(`_`)
			delete(unused, name)
		}
	}
	if len(unused) == 0 {
		return
	}

	// Remove all unused import specifications
	isUnusedSpec := map[*ast.ImportSpec]bool{}
	for _, index := range unused {
		isUnusedSpec[file.Imports[index]] = true
	}
	for _, decl := range file.Decls {
		if d, ok := decl.(*ast.GenDecl); ok {
			for i, spec := range d.Specs {
				if other, ok := spec.(*ast.ImportSpec); ok && isUnusedSpec[other] {
					d.Specs[i] = nil
				}
			}
		}
	}

	// Remove the unused import copies in the file
	for _, index := range unused {
		file.Imports[index] = nil
	}

	FinalizeRemovals(file)
}

// squeeze removes all nil nodes from the slice.
//
// The given slice will be modified. This is designed for squeezing
// declaration, specification, imports, and identifier lists.
func squeeze[E ast.Node, S ~[]E](s S) S {
	var zero E
	count, dest := len(s), 0
	for src := 0; src < count; src++ {
		if !reflect.DeepEqual(s[src], zero) {
			// Swap the values, this will put the nil values to the end
			// of the slice so that the tail isn't holding onto pointers.
			s[dest], s[src] = s[src], s[dest]
			dest++
		}
	}
	return s[:dest]
}

// updateFileComments rebuilds the file comments by reading the comments
// off of the nodes in the file. Any comments that are not associated with
// a node will be lost.
func updateFileComments(file *ast.File) {
	file.Comments = nil // clear this first so ast.Inspect doesn't walk it.
	remComments := []*ast.CommentGroup{}
	ast.Inspect(file, func(n ast.Node) bool {
		if cg, ok := n.(*ast.CommentGroup); ok {
			remComments = append(remComments, cg)
		}
		return true
	})
	file.Comments = remComments
}

// FinalizeRemovals fully removes any declaration, specification, imports
// that have been set to nil. This will also remove any unassociated comment
// groups, including the comments from removed code.
// Comments that are floating and tied to a node will be lost.
func FinalizeRemovals(file *ast.File) {
	fileChanged := false
	for i, decl := range file.Decls {
		switch d := decl.(type) {
		case nil:
			fileChanged = true
		case *ast.GenDecl:
			declChanged := false
			for j, spec := range d.Specs {
				switch s := spec.(type) {
				case nil:
					declChanged = true
				case *ast.ValueSpec:
					specChanged := false
					for _, name := range s.Names {
						if name == nil {
							specChanged = true
							break
						}
					}
					if specChanged {
						s.Names = squeeze(s.Names)
						s.Values = squeeze(s.Values)
						if len(s.Names) == 0 {
							declChanged = true
							d.Specs[j] = nil
						}
					}
				}
			}
			if declChanged {
				d.Specs = squeeze(d.Specs)
				if len(d.Specs) == 0 {
					fileChanged = true
					file.Decls[i] = nil
				}
			}
		}
	}
	if fileChanged {
		file.Decls = squeeze(file.Decls)
	}

	file.Imports = squeeze(file.Imports)

	updateFileComments(file)
}

// ConcatenateFiles will concatenate the given tailing files onto the
// end of the first given AST file.
//
// This is designed to handle concatenating native overrides into the original
// source files so won't work for general purpose AST file concatenation.
//
// Returns an error if the concatenation fails.
//
// Caveats:
//   - The Pos fields will not be modified so that that source locations will
//     still show the correct file or virtual file positions.
//   - The given file will be modified even if an error is returned and may
//     be in an invalid state.
//   - The tail files must be from the same package name and have the same import
//     names for imports with the same import path.
//   - Any duplicate objects must have been already resolved via an overlay
//     augmentation prior to concatenation so that there are no duplicate objects.
//     Any remaining duplicate objects will cause an error to be returned.
//   - The tails will not be modified, however the nodes from the tails will be
//     added into the target file so modifications to the tails after
//     concatenation could cause the target file to be in an invalid state.
//   - This will not modify the deprecated Unresolved or file Scope fields.
//   - Any comments on import declarations will be lost since the imports will
//     be merged into a single new import declaration. The comments on the
//     individual import specs will be preserved.
//   - The package comments will be concatenated. It will not check for
//     build constraints or any file level directives, but simply append
//     the tail comments as is. This may cause issues when formatting
//     the resulting file including extra newlines or invalid code.
func ConcatenateFiles(file *ast.File, tails ...*ast.File) error {
	// Populate the imports map from the target file.
	// This map will be used to check for duplicate imports.
	imports := make(map[string]*ast.ImportSpec, len(file.Imports))
	for _, imp := range file.Imports {
		imports[imp.Path.Value] = imp
	}

	// Get list of declarations not including the imports.
	decls := make([]ast.Decl, 0, len(file.Decls))
	for _, decl := range file.Decls {
		if gen, ok := decl.(*ast.GenDecl); !ok || gen.Tok != token.IMPORT {
			decls = append(decls, decl)
		}
	}

	// Merge in all the tail files into the target file.
	for _, tail := range tails {

		// Check the package names match.
		if file.Name.Name != tail.Name.Name {
			return fmt.Errorf("can not concatenate files with different package names: %q != %q", file.Name.Name, tail.Name.Name)
		}

		// Concatenate the imports.
		for _, imp := range tail.Imports {
			path := imp.Path.Value
			if oldImp, ok := imports[path]; ok {
				// Import is in both files so check if the import name is not different.
				oldName, newName := ImportName(oldImp), ImportName(imp)
				if oldName != newName {
					if len(oldName) == 0 {
						// Update the import name to the new name.
						// This assumes the import name was `_` and
						// could cause problems if it was `.`
						oldImp.Name = imp.Name
					} else if len(newName) != 0 {
						return fmt.Errorf("import from of %s can not be concatenated with different name: %q != %q", path, oldName, newName)
					}
				}
				continue
			}
			imports[imp.Path.Value] = imp
		}

		// Concatenate the declarations while skipping imports.
		for _, decl := range tail.Decls {
			if gen, ok := decl.(*ast.GenDecl); !ok || gen.Tok != token.IMPORT {
				decls = append(decls, decl)
			}
		}

		// Concatenate the document comments.
		if tail.Doc != nil {
			if file.Doc == nil {
				file.Doc = &ast.CommentGroup{}
			}
			file.Doc.List = append(file.Doc.List, tail.Doc.List...)

			// To help prevent issues when formatting causing a document comment
			// to occur between `package` and the package name, move the package
			// name to the Pos of the tail so it comes after the tail's package comment.
			file.Package = tail.Package
			file.Name.NamePos = tail.Name.NamePos
		}
	}

	// Update the target file's declarations with all the imports
	// prepended to the list of declarations as one import declaration.
	// Also sort the imports by path to ensure a consistent order.
	if len(imports) > 0 {
		importsGen := &ast.GenDecl{
			Tok:   token.IMPORT,
			Specs: make([]ast.Spec, 0, len(file.Imports)),
		}
		paths := make([]string, 0, len(imports))
		for path := range imports {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		file.Imports = make([]*ast.ImportSpec, 0, len(imports))
		for _, path := range paths {
			imp := imports[path]
			importsGen.Specs = append(importsGen.Specs, imp)
			file.Imports = append(file.Imports, imp)
		}
		decls = append([]ast.Decl{importsGen}, decls...)
	}
	file.Decls = decls

	updateFileComments(file)
	return nil
}
