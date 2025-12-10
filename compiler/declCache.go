package compiler

import (
	"fmt"
	"sort"
)

type DeclCache struct {
	decls   map[string]*Decl
	changed bool
}

// Chacnged reports whether the cache has had declarations added to it since
// it was created or last read from storage or created empty.
//
// If there have been no changes, the cache does not need to be written back
// to storage. Typically declarations will only be added when the cache is
// empty and not loaded from storage since otherwise all the needed declarations
// (other than those not being cached) would already be present.
func (dc *DeclCache) Changed() bool {
	return dc != nil && dc.changed
}

func (dc *DeclCache) GetDecl(fullname string) *Decl {
	if dc == nil {
		return nil // cache is disabled
	}
	return dc.decls[fullname]
}

func (dc *DeclCache) PutDecl(decl *Decl) {
	if dc == nil {
		return // cache is disabled
	}

	if decl.ForGeneric {
		// Do not cache declarations for generic instantiations.
		// The type arguments may come from a package depending on this one
		// and not on one of this package's dependencies.
		//
		// If one of this package's dependencies changes, the cache will not be used.
		// However, currently, changes to packages depending on this one
		// may change and this package's cache may still be used.
		// Therefore, if the package depending on this one changes a type from
		// being blocking to non-blocking or vice versa, the cached declaration
		// may be invalid.
		return
	}

	if isUnqueDeclFullName(decl.FullName) {
		// Only cache declarations with unique names.
		return
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

func (dc *DeclCache) Read(decode func(any) error) error {
	if dc == nil {
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
		dc.decls[decl.FullName] = decl
	}
	return nil
}

func (dc *DeclCache) Write(encode func(any) error) error {
	if dc == nil {
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
		if err := encode(dc.decls[name]); err != nil {
			return err
		}
	}
	return nil
}
