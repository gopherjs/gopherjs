// +build go1.16

package build

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"github.com/visualfc/goembed"
)

func buildIdent(name string) string {
	return fmt.Sprintf("__js_embed_%x__", name)
}

var embed_head = `package $pkg

import (
	"embed"
	_ "unsafe"
)

//go:linkname gopherjs_embed_append embed.appendData
func gopherjs_embed_append(fs *embed.FS, name string, data string, hash [16]byte)

`

func checkEmbed(pkg *PackageData, embedPatternPos map[string][]token.Position, fset *token.FileSet, files []*ast.File, outPkgName string, outPkgFile string) (*ast.File, error) {
	if len(embedPatternPos) == 0 {
		return nil, nil
	}
	ems := goembed.CheckEmbed(embedPatternPos, fset, files)
	r := goembed.NewResolve()
	var buf bytes.Buffer
	buf.WriteString(strings.Replace(embed_head, "$pkg", outPkgName, 1))
	buf.WriteString("\nfunc init() {\n")
	for _, v := range ems {
		fs, _ := r.Load(pkg.Dir, v)
		switch v.Kind {
		case goembed.EmbedBytes:
			buf.WriteString(fmt.Sprintf("\t%v = %v\n", v.Name, buildIdent(fs[0].Name)))
		case goembed.EmbedString:
			buf.WriteString(fmt.Sprintf("\t%v = string(%v)\n", v.Name, buildIdent(fs[0].Name)))
		case goembed.EmbedFiles:
			fs = goembed.BuildFS(fs)
			for _, f := range fs {
				if len(f.Data) == 0 {
					buf.WriteString(fmt.Sprintf("\tgopherjs_embed_append(&%v,\"%v\",\"\",[16]byte{})\n", v.Name, f.Name))
				} else {
					buf.WriteString(fmt.Sprintf("\tgopherjs_embed_append(&%v,\"%v\",string(%v),[16]byte{})\n", v.Name, f.Name, buildIdent(f.Name)))
				}
			}
		}
	}
	buf.WriteString("\n}\n")
	buf.WriteString("\nvar (\n")
	for _, f := range r.Files() {
		if len(f.Data) > 0 {
			hex, _ := goembed.BytesToHex(f.Data)
			buf.WriteString(fmt.Sprintf("\t%v = []byte(\"%v\")\n", buildIdent(f.Name), hex))
		} else {
			buf.WriteString(fmt.Sprintf("\t%v []byte\n", buildIdent(f.Name)))
		}
	}
	buf.WriteString(")\n")
	f, err := parser.ParseFile(fset, outPkgFile, buf.String(), parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (s *Session) checkEmbed(pkg *PackageData, fset *token.FileSet, files []*ast.File) ([]*ast.File, error) {
	var embeds []*ast.File
	if len(pkg.EmbedPatternPos) > 0 {
		f, err := checkEmbed(pkg, pkg.EmbedPatternPos, fset, files, pkg.Name, "js_embed.go")
		if err != nil {
			return nil, err
		}
		embeds = append(embeds, f)
	}
	if len(pkg.TestEmbedPatternPos) > 0 {
		f, err := checkEmbed(pkg, pkg.TestEmbedPatternPos, fset, files, pkg.Name, "js_embed_test.go")
		if err != nil {
			return nil, err
		}
		embeds = append(embeds, f)
	}
	if len(pkg.XTestEmbedPatternPos) > 0 {
		f, err := checkEmbed(pkg, pkg.XTestEmbedPatternPos, fset, files, pkg.Name+"_test", "js_embed_x_test.go")
		if err != nil {
			return nil, err
		}
		embeds = append(embeds, f)
	}
	return embeds, nil
}
