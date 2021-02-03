// +build go1.16

package build

import (
	"go/ast"
	"go/token"
	"log"
	"sort"
	"strings"
)

type Embed struct {
	Name     string
	Kind     int
	Patterns []string
}

type embedPattern struct {
	Patterns string
	Pos      token.Position
}

type embedPatterns struct {
	Patterns []string
	Pos      token.Position
}

func (s *Session) checkEmbed(pkg *PackageData, fset *token.FileSet, files []*ast.File) error {
	if len(pkg.EmbedPatternPos) == 0 {
		return nil
	}
	fmap := make(map[string]bool)
	var ep []*embedPattern
	for k, v := range pkg.EmbedPatternPos {
		for _, pos := range v {
			fmap[pos.Filename] = true
			ep = append(ep, &embedPattern{k, pos})
		}
	}
	sort.SliceStable(ep, func(i, j int) bool {
		n := strings.Compare(ep[i].Pos.Filename, ep[j].Pos.Filename)
		if n == 0 {
			return ep[i].Pos.Offset < ep[j].Pos.Offset
		}
		return n < 0
	})
	var eps []*embedPatterns
	last := &embedPatterns{[]string{ep[0].Patterns}, ep[0].Pos}
	eps = append(eps, last)
	for i := 1; i < len(ep); i++ {
		e := ep[i]
		if e.Pos.Filename == last.Pos.Filename &&
			e.Pos.Line == last.Pos.Line+1 {
			last.Patterns = append(last.Patterns, e.Patterns)
			last.Pos = e.Pos
		} else {
			last = &embedPatterns{[]string{e.Patterns}, e.Pos}
			eps = append(eps, last)
		}
	}
	var embeds []*Embed
	for _, file := range files {
		if fmap[fset.Position(file.Package).Filename] {
			ems := findEmbed(pkg, fset, file, eps)
			if len(ems) > 0 {
				embeds = append(embeds, ems...)
			}
		}
	}
	for _, e := range embeds {
		log.Println(e)
	}
	return nil
}

const (
	embedUnknown int = iota
	embedBytes
	embedString
	embedFiles
)

func checkIdent(v ast.Expr, name string) bool {
	if ident, ok := v.(*ast.Ident); ok && ident.Name == name {
		return true
	}
	return false
}

func embedKind(typ ast.Expr) int {
	switch v := typ.(type) {
	case *ast.Ident:
		if checkIdent(v, "string") {
			return embedString
		}
	case *ast.ArrayType:
		if checkIdent(v.Elt, "byte") {
			return embedBytes
		}
	case *ast.SelectorExpr:
		if checkIdent(v.X, "embed") && checkIdent(v.Sel, "FS") {
			return embedFiles
		}
	}
	return embedUnknown
}

func findEmbed(pkg *PackageData, fset *token.FileSet, file *ast.File, eps []*embedPatterns) (embeds []*Embed) {
	for _, decl := range file.Decls {
		if d, ok := decl.(*ast.GenDecl); ok && d.Tok == token.VAR {
			pos := fset.Position(d.Pos())
			for _, e := range eps {
				if pos.Filename == e.Pos.Filename &&
					pos.Line == e.Pos.Line+1 {
					if len(d.Specs) == 1 {
						if spec, ok := d.Specs[0].(*ast.ValueSpec); ok {
							embeds = append(embeds,
								&Embed{
									Name:     spec.Names[0].Name,
									Kind:     embedKind(spec.Type),
									Patterns: e.Patterns,
								},
							)
						}
					}
				}
			}
		}
	}
	return
}
