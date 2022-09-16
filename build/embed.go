package build

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"

	"github.com/visualfc/goembed"
)

func buildIdent(name string) string {
	return fmt.Sprintf("__gopherjs_embed_%x__", name)
}

var embed_head = `package %v

import (
	"embed"
	_ "unsafe"
)

//go:linkname __gopherjs_embed_buildFS__ embed.buildFS
func __gopherjs_embed_buildFS__(list []struct {
	name string
	data string
	hash [16]byte
}) (f embed.FS)
`

// embedFiles generates an additional source file, which initializes all variables in the package with a go:embed directive.
func embedFiles(bp *PackageData, fset *token.FileSet, files []*ast.File) (*ast.File, error) {
	if len(bp.EmbedPatternPos) == 0 {
		return nil, nil
	}

	ems, err := goembed.CheckEmbed(bp.EmbedPatternPos, fset, files)
	if err != nil {
		return nil, err
	}

	r := goembed.NewResolve()
	for _, v := range ems {
		fs, err := r.Load(bp.Dir, fset, v)
		if err != nil {
			return nil, err
		}
		switch v.Kind {
		case goembed.EmbedMaybeAlias:
			// value = Type(data)
			// valid alias string or []byte type used by types.check
			v.Spec.Values = []ast.Expr{
				&ast.CallExpr{
					Fun: v.Spec.Type,
					Args: []ast.Expr{
						&ast.Ident{Name: buildIdent(fs[0].Name),
							NamePos: v.Spec.Names[0].NamePos},
					},
				}}
		case goembed.EmbedBytes:
			// value = []byte(data)
			v.Spec.Values = []ast.Expr{
				&ast.CallExpr{
					Fun:  v.Spec.Type,
					Args: []ast.Expr{ast.NewIdent(buildIdent(fs[0].Name))},
				}}
		case goembed.EmbedString:
			// value = data
			v.Spec.Values = []ast.Expr{ast.NewIdent(buildIdent(fs[0].Name))}
		case goembed.EmbedFiles:
			// value = __gopherjs_embed_buildFS__([]struct{name string; data string; hash [16]byte}{...})
			fs = goembed.BuildFS(fs)
			elts := make([]ast.Expr, len(fs))
			for i, f := range fs {
				if len(f.Data) == 0 {
					elts[i] = &ast.CompositeLit{
						Elts: []ast.Expr{
							&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(f.Name)},
							&ast.BasicLit{Kind: token.STRING, Value: `""`},
							&ast.CompositeLit{
								Type: &ast.ArrayType{
									Len: &ast.BasicLit{Kind: token.INT, Value: "16"},
									Elt: ast.NewIdent("byte"),
								},
							},
						},
					}
				} else {
					var hash [16]ast.Expr
					for j, v := range f.Hash {
						hash[j] = &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(int(v))}
					}
					elts[i] = &ast.CompositeLit{
						Elts: []ast.Expr{
							&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(f.Name)},
							ast.NewIdent(buildIdent(f.Name)),
							&ast.CompositeLit{
								Type: &ast.ArrayType{
									Len: &ast.BasicLit{Kind: token.INT, Value: "16"},
									Elt: ast.NewIdent("byte"),
								},
								Elts: hash[:],
							},
						},
					}
				}
			}
			call := &ast.CallExpr{
				Fun: ast.NewIdent("__gopherjs_embed_buildFS__"),
				Args: []ast.Expr{
					&ast.CompositeLit{
						Type: &ast.ArrayType{
							Elt: &ast.StructType{
								Fields: &ast.FieldList{
									List: []*ast.Field{
										&ast.Field{
											Names: []*ast.Ident{ast.NewIdent("name")},
											Type:  ast.NewIdent("string"),
										},
										&ast.Field{
											Names: []*ast.Ident{ast.NewIdent("data")},
											Type:  ast.NewIdent("string"),
										},
										&ast.Field{
											Names: []*ast.Ident{ast.NewIdent("hash")},
											Type: &ast.ArrayType{
												Len: &ast.BasicLit{Kind: token.INT, Value: "16"},
												Elt: ast.NewIdent("byte"),
											},
										},
									},
								},
							},
						},
						Elts: elts,
					},
				},
			}
			v.Spec.Values = []ast.Expr{call}
		}
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, embed_head, bp.Name)
	buf.WriteString("\nconst (\n")
	for _, f := range r.Files() {
		if len(f.Data) == 0 {
			fmt.Fprintf(&buf, "\t%v = \"\"\n", buildIdent(f.Name))
		} else {
			fmt.Fprintf(&buf, "\t%v = \"%v\"\n", buildIdent(f.Name), goembed.BytesToHex(f.Data))
		}
	}
	buf.WriteString(")\n\n")
	f, err := parser.ParseFile(fset, "js_embed.go", buf.String(), parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return f, nil
}
