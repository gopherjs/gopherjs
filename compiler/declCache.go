package compiler

import (
	"encoding/gob"
	"fmt"
	"sort"
)

type DeclCache struct {
	enabled bool
	decls   map[string]*Decl
	changed bool
}

func init() {
	// Register any types that are referenced by an interface so that
	// the gob encoder/decoder can handle them.
	gob.Register(&DeclCache{})
}

func NewDeclCache(enabled bool) *DeclCache {
	return &DeclCache{enabled: enabled}
}

func (dc *DeclCache) GetDecl(fullname string) *Decl {
	if dc == nil || !dc.enabled {
		return nil // cache is disabled
	}
	return dc.decls[fullname]
}

func (dc *DeclCache) PutDecl(decl *Decl) {
	if dc == nil || !dc.enabled {
		return // cache is disabled
	}

	if dc.decls == nil {
		dc.decls = map[string]*Decl{}
	}
	if existing, ok := dc.decls[decl.FullName]; ok {
		if existing != decl {
			panic(fmt.Errorf(`decl cache conflict: different decls with same name: %q`, decl.FullName))
		}
		return
	}
	dc.decls[decl.FullName] = decl
	dc.changed = true
}

func (dc *DeclCache) Changed() bool {
	return dc != nil && dc.changed
}

func (dc *DeclCache) Read(decode func(any) error) error {
	if dc == nil || !dc.enabled {
		return nil // cache is disabled
	}

	var count int
	if err := decode(&count); err != nil {
		return err
	}

	if dc.decls == nil {
		dc.decls = map[string]*Decl{}
	}
	for i := 0; i < count; i++ {
		decl := &Decl{}
		if err := decode(decl); err != nil {
			return err
		}
		fmt.Printf("Read decl from cache: %s\n", decl.FullName) // TODO(grantnelson-wf): REMOVE
		dc.decls[decl.FullName] = decl
	}

	return nil
}

func (dc *DeclCache) Write(encode func(any) error) error {
	if dc == nil || !dc.enabled {
		return nil // cache is disabled
	}

	count := len(dc.decls)
	if err := encode(count); err != nil {
		return err
	}

	names := make([]string, 0, count)
	for name := range dc.decls {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Printf("Wrote decl to cache: %s\n", name) // TODO(grantnelson-wf): REMOVE
		if err := encode(dc.decls[name]); err != nil {
			return err
		}
	}
	return nil
}
