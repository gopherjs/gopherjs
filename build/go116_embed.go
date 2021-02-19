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

//go:linkname gopherjs_embed_buildFS embed.buildFS
func gopherjs_embed_buildFS(list []struct {
	name string
	data string
	hash [16]byte
}) (f embed.FS)
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
	buf.WriteString("\nvar (\n")
	for _, v := range ems {
		v.Spec.Names[0].Name = "_"
		fs, _ := r.Load(pkg.Dir, v)
		switch v.Kind {
		case goembed.EmbedBytes:
			buf.WriteString(fmt.Sprintf("\t%v = []byte(%v)\n", v.Name, buildIdent(fs[0].Name)))
		case goembed.EmbedString:
			buf.WriteString(fmt.Sprintf("\t%v = %v\n", v.Name, buildIdent(fs[0].Name)))
		case goembed.EmbedFiles:
			fs = goembed.BuildFS(fs)
			buf.WriteString(fmt.Sprintf("\t%v=", v.Name))
			buf.WriteString(`gopherjs_embed_buildFS(
[]struct {
	name string
	data string
	hash [16]byte
}{
`)
			for _, f := range fs {
				if len(f.Data) == 0 {
					buf.WriteString(fmt.Sprintf("\t{\"%v\",\"\",[16]byte{}},\n",
						f.Name))
				} else {
					buf.WriteString(fmt.Sprintf("\t{\"%v\",%v,[16]byte{%v}},\n",
						f.Name, buildIdent(f.Name), goembed.BytesToList(f.Hash[:])))
				}
			}
			buf.WriteString("})\n")
		}
	}
	buf.WriteString("\n)\n")
	buf.WriteString("\nvar (\n")
	for _, f := range r.Files() {
		if len(f.Data) == 0 {
			buf.WriteString(fmt.Sprintf("\t%v string\n",
				buildIdent(f.Name)))
		} else {
			buf.WriteString(fmt.Sprintf("\t%v = string(\"%v\")\n",
				buildIdent(f.Name), goembed.BytesToHex(f.Data)))
		}
	}
	buf.WriteString(")\n\n")
	f, err := parser.ParseFile(fset, "js_embed.go", buf.String(), parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return f, nil
}
