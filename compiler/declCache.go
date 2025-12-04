package compiler

type DeclCache struct {
	enabled bool
	// TODO: Implement decl cache.
}

func NewDeclCache(enable bool) *DeclCache {
	return &DeclCache{
		enabled: enable,
	}
}

// TODO: May need a more unique key since some decls may not have unique names.
func (dc *DeclCache) GetDecl(fullname string) *Decl {
	return nil
}

func (dc *DeclCache) PutDecl(decl *Decl) {

}

func (dc *DeclCache) Read(decode func(any) error) error {
	// TODO: Implement decl cache serialization.
	return nil
}

func (dc *DeclCache) Write(encode func(any) error) error {
	// TODO: Implement decl cache serialization.
	return nil
}
