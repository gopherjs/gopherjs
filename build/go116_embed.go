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

func (s *Session) checkEmbed(pkg *PackageData, fset *token.FileSet, files []*ast.File) (*ast.File, error) {
	if len(pkg.EmbedPatternPos) == 0 {
		return nil, nil
	}
	ems := goembed.CheckEmbed(pkg.EmbedPatternPos, fset, files)
	if len(ems) == 0 {
		return nil, nil
	}
	r := goembed.NewResolve()
	var buf bytes.Buffer
	buf.WriteString(strings.Replace(embed_head, "$pkg", pkg.Name, 1))
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
	f, err := parser.ParseFile(fset, "js_embed.go", buf.String(), parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return f, nil
}
