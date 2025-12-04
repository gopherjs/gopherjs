package compiler

import (
	"errors"
	"fmt"
	"time"
)

type DeclCache struct {
	importPath    string
	importedPaths []string
	importDecls   []*Decl
	typeDecls     []*Decl
	varDecls      []*Decl
	funcDecls     []*Decl

	cacheLoaded bool
	changed     bool
	stats       declCacheStats
}

// NewDeclCache creates a new empty enabled decl cache for the package with
// the given import path.
//
// The given import path is used as a secondary check to ensure we do not get
// hashing collisions when reading a cache from storage.
func NewDeclCache(importPath string) *DeclCache {
	return &DeclCache{importPath: importPath}
}

var declCacheGlobalStats declCacheStats

type declCacheStats struct {
	reads      int // number of times Read was called (should be 0 or 1 unless global)
	writes     int // number of times Write was called (should be 0 or 1 unless global)
	readCount  int // number of declarations loaded from storage
	writeCount int // number of declarations written to storage
	readDur    time.Duration
	writeDur   time.Duration
}

var (
	errReadDisabled  = errors.New(`attempted to read a DeclCache that is disabled (nil)`)
	errWriteDisabled = errors.New(`attempted to write a DeclCache that is disabled (nil)`)
)

// ImportPath is the import path for the package this cache is for.
func (dc *DeclCache) ImportPath() string {
	if dc == nil {
		return `[disabled]`
	}
	return dc.importPath
}

// Changed reports whether the cache has had declarations set to it since
// it was created (or last read from or write to storage). If there have been
// no changes, the cache does not need to be written back to storage.
func (dc *DeclCache) Changed() bool {
	return dc != nil && dc.changed
}

// HasCache indicates that this cache has been loaded from the cache file.
func (dc *DeclCache) HasCache() bool {
	return dc != nil && dc.cacheLoaded
}

// count reports how many declarations are currently in the cache.
func (dc *DeclCache) count() int {
	if dc == nil {
		return 0
	}
	return len(dc.importDecls) + len(dc.typeDecls) + len(dc.varDecls) + len(dc.funcDecls)
}

func (dc *DeclCache) GetDecls() (importedPaths []string, importDecls, typeDecls, varDecls, funcDecls []*Decl) {
	if dc == nil {
		return nil, nil, nil, nil, nil // cache is disabled
	}
	return dc.importedPaths, dc.importDecls, dc.typeDecls, dc.varDecls, dc.funcDecls
}

func (dc *DeclCache) SetDecls(importedPaths []string, importDecls, typeDecls, varDecls, funcDecls []*Decl) {
	if dc == nil {
		return // cache is disabled
	}
	dc.importedPaths = importedPaths
	dc.importDecls = importDecls
	dc.typeDecls = typeDecls
	dc.varDecls = varDecls
	dc.funcDecls = funcDecls
	dc.changed = true
}

type serializableDeclCache struct {
	ImportPath    string   `json:"path"`
	ImportedPaths []string `json:"p,omitempty"`
	ImportDecls   []*Decl  `json:"i,omitempty"`
	TypeDecls     []*Decl  `json:"t,omitempty"`
	VarDecls      []*Decl  `json:"v,omitempty"`
	FuncDecls     []*Decl  `json:"f,omitempty"`
}

func (dc *DeclCache) Read(decode func(any) error) error {
	if dc == nil {
		return errReadDisabled
	}

	start := time.Now()

	ser := serializableDeclCache{}
	if err := decode(&ser); err != nil {
		return err
	}
	if dc.importPath != ser.ImportPath {
		return fmt.Errorf(`read cache for import path %q when wanting %q`, dc.importPath, ser.ImportPath)
	}

	// Only modify the cache after everything has been checked.
	dc.importedPaths = ser.ImportedPaths
	dc.importDecls = ser.ImportDecls
	dc.typeDecls = ser.TypeDecls
	dc.varDecls = ser.VarDecls
	dc.funcDecls = ser.FuncDecls
	dc.changed = false
	dc.cacheLoaded = true

	dc.stats.reads++
	declCacheGlobalStats.reads++

	declCount := dc.count()
	dc.stats.readCount += declCount
	declCacheGlobalStats.readCount += declCount

	dur := time.Since(start)
	dc.stats.readDur += dur
	declCacheGlobalStats.readDur += dur
	return nil
}

func (dc *DeclCache) Write(encode func(any) error) error {
	if dc == nil {
		return errWriteDisabled
	}

	start := time.Now()

	ser := serializableDeclCache{
		ImportedPaths: dc.importedPaths,
		ImportDecls:   dc.importDecls,
		ImportPath:    dc.importPath,
		TypeDecls:     dc.typeDecls,
		VarDecls:      dc.varDecls,
		FuncDecls:     dc.funcDecls,
	}
	if err := encode(ser); err != nil {
		return err
	}
	dc.changed = false

	dc.stats.writes++
	declCacheGlobalStats.writes++

	declCount := dc.count()
	dc.stats.writeCount += declCount
	declCacheGlobalStats.writeCount += declCount

	dur := time.Since(start)
	dc.stats.writeDur += dur
	declCacheGlobalStats.writeDur += dur
	return nil
}

func (dc *DeclCache) String() string {
	if dc == nil {
		return `[disabled]`
	}
	return dc.stats.String() + `, ` + dc.importPath
}

func (s declCacheStats) String() string {
	return fmt.Sprintf(`reads: %d, writes: %d, readCount: %d, writeCount: %d, readDur: %f sec, writeDur: %f sec`,
		s.reads, s.writes, s.readCount, s.writeCount, s.readDur.Seconds(), s.writeDur.Seconds())
}

func GlobalDeclCacheStats() string {
	return declCacheGlobalStats.String() + `, [global]`
}
