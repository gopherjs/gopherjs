// +build !go1.16

package build

import (
	"go/ast"
	"go/token"
)

func (s *Session) checkEmbed(pkg *PackageData, fileSet *token.FileSet, files []*ast.File) (*ast.File, error) {
	return nil, nil
}
