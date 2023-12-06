package astutil

import (
	"fmt"
	"go/ast"
	"regexp"
)

// PruneOriginal returns true if gopherjs:prune-original directive is present
// before a function decl.
//
// `//gopherjs:prune-original` is a GopherJS-specific directive, which can be
// applied to functions in native overlays and will instruct the augmentation
// logic to delete the body of a standard library function that was replaced.
// This directive can be used to remove code that would be invalid in GopherJS,
// such as code expecting ints to be 64-bit. It should be used with caution
// since it may create unused imports in the original source file.
func PruneOriginal(d *ast.FuncDecl) bool {
	return hasDirective(d, `prune-original`)
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

// PurgeOriginal returns true if gopherjs:purge-original directive is present
// on a struct, interface, variables, constants, and functions.
//
// `//gopherjs:purge-original` is a GopherJS-specific directive, which can be
// applied in native overlays and will instruct the augmentation logic to
// delete part of the standard library without a replacement. This directive
// can be used to remove code that would be invalid in GopherJS, such as code
// using unsupported features (e.g. generic interfaces and methods before
// generics were fully supported).
// It should be used with caution since it may remove needed dependencies.
func PurgeOriginal(d any) bool {
	return hasDirective(d, `purge-original`)
}

// anyDocLine calls the given predicate on all associated documentation
// lines and line-comment lines from the given node.
// If the predicate returns true for any line then true is returned.
func anyDocLine(node any, predicate func(line string) bool) bool {
	switch a := node.(type) {
	case *ast.Comment:
		return a != nil && predicate(a.Text)
	case *ast.CommentGroup:
		if a != nil {
			for _, c := range a.List {
				if anyDocLine(c, predicate) {
					return true
				}
			}
		}
		return false
	case *ast.Field:
		return a != nil && (anyDocLine(a.Doc, predicate) || anyDocLine(a.Comment, predicate))
	case *ast.File:
		return a != nil && anyDocLine(a.Doc, predicate)
	case *ast.FuncDecl:
		return a != nil && anyDocLine(a.Doc, predicate)
	case *ast.GenDecl:
		return a != nil && anyDocLine(a.Doc, predicate)
	case *ast.ImportSpec:
		return a != nil && (anyDocLine(a.Doc, predicate) || anyDocLine(a.Comment, predicate))
	case *ast.TypeSpec:
		return a != nil && (anyDocLine(a.Doc, predicate) || anyDocLine(a.Comment, predicate))
	case *ast.ValueSpec:
		return a != nil && (anyDocLine(a.Doc, predicate) || anyDocLine(a.Comment, predicate))
	default:
		panic(fmt.Errorf(`unexpected node type to get doc from: %T`, node))
	}
}

// directiveMatcher is a regex which matches a GopherJS directive
// and finds the directive action.
//
// This matches the largest directive action until whitespace or EOL
// to differentiate from any directive action which is a prefix
// for another directive action.
var directiveMatcher = regexp.MustCompile(`^\/(?:\/|\*)gopherjs:([\w-]+)`)

// hasDirective returns true if the associated documentation
// or line comments for the given node have the given directive action.
//
// All gopherjs directives must start with `//gopherjs:` or `/*gopherjs:`
// and followed by an action without any whitespace. The action must be
// one or more letter, decimal, underscore, or hyphen.
//
// see https://pkg.go.dev/cmd/compile#hdr-Compiler_Directives
func hasDirective(node any, directive string) bool {
	return anyDocLine(node, func(line string) bool {
		m := directiveMatcher.FindStringSubmatch(line)
		return len(m) == 2 && m[1] == directive
	})
}
