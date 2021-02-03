// +build !go1.16

package build

func (s *Session) checkEmbed(pkg *PackageData, fileSet *token.FileSet, files []*ast.File) error {
	return nil
}
