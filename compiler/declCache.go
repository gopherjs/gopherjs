package compiler

import (
	"encoding/gob"
	"fmt"
)

type DeclCache struct {
	enabled bool
	decls   map[string]*Decl
}

func init() {
	gob.Register(&DeclCache{})
}

func NewDeclCache(enable bool) *DeclCache {
	return &DeclCache{
		enabled: enable,
		decls:   map[string]*Decl{},
	}
}

func (dc *DeclCache) GetDecl(fullname string) *Decl {
	return dc.decls[fullname]
}

func (dc *DeclCache) PutDecl(decl *Decl) {
	if !dc.enabled {
		// If not enabled, do nothing since the cache is not being stored.
		return
	}

	if existing, ok := dc.decls[decl.FullName]; ok {
		if existing != decl {
			panic(fmt.Errorf(`decl cache conflict: different decls with same name: %q`, decl.FullName))
		}
		return
	}
	dc.decls[decl.FullName] = decl
}

func (dc *DeclCache) Read(decode func(any) error) error {
	// TODO: Implement decl cache serialization.
	return nil
}

func (dc *DeclCache) Write(encode func(any) error) error {
	// TODO: Implement decl cache serialization.
	return nil
}
