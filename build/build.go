package build

import (
	"bitbucket.org/kardianos/osext"
	"code.google.com/p/go.exp/fsnotify"
	"fmt"
	"github.com/gopherjs/gopherjs/compiler"
	"github.com/neelance/sourcemap"
	"go/ast"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type ImportCError struct{}

func (e *ImportCError) Error() string {
	return `importing "C" is not supported by GopherJS`
}

func NewBuildContext(archSuffix string) *build.Context {
	return &build.Context{
		GOROOT:    build.Default.GOROOT,
		GOPATH:    build.Default.GOPATH,
		GOOS:      build.Default.GOOS,
		GOARCH:    archSuffix,
		Compiler:  "gc",
		BuildTags: []string{"netgo", runtime.Version()[:5]},
	}
}

func Import(path string, mode build.ImportMode, archSuffix string) (*build.Package, error) {
	if path == "C" {
		return nil, &ImportCError{}
	}

	buildContext := NewBuildContext(archSuffix)
	if path == "runtime" || path == "syscall" {
		buildContext.GOARCH = build.Default.GOARCH
		buildContext.InstallSuffix = archSuffix
	}
	pkg, err := buildContext.Import(path, "", mode)
	if path == "hash/crc32" {
		pkg.GoFiles = []string{"crc32.go", "crc32_generic.go"}
	}
	if pkg.IsCommand() {
		pkg.PkgObj = filepath.Join(pkg.BinDir, filepath.Base(pkg.ImportPath)+".js")
	}
	if _, err := os.Stat(pkg.PkgObj); os.IsNotExist(err) && strings.HasPrefix(pkg.PkgObj, build.Default.GOROOT) {
		// fall back to GOPATH
		firstGopathWorkspace := filepath.SplitList(build.Default.GOPATH)[0] // TODO: Need to check inside all GOPATH workspaces.
		gopathPkgObj := filepath.Join(firstGopathWorkspace, pkg.PkgObj[len(build.Default.GOROOT):])
		if _, err := os.Stat(gopathPkgObj); err == nil {
			pkg.PkgObj = gopathPkgObj
		}
	}
	return pkg, err
}

func Parse(pkg *build.Package, fileSet *token.FileSet) ([]*ast.File, error) {
	var files []*ast.File
	replacedDeclNames := make(map[string]bool)
	funcName := func(d *ast.FuncDecl) string {
		if d.Recv == nil {
			return d.Name.Name
		}
		recv := d.Recv.List[0].Type
		if star, ok := recv.(*ast.StarExpr); ok {
			recv = star.X
		}
		return recv.(*ast.Ident).Name + "." + d.Name.Name
	}
	isTestPkg := strings.HasSuffix(pkg.ImportPath, "_test")
	importPath := pkg.ImportPath
	if isTestPkg {
		importPath = importPath[:len(importPath)-5]
	}
	if nativesPkg, err := Import("github.com/gopherjs/gopherjs/compiler/natives/"+importPath, 0, "js"); err == nil {
		names := append(nativesPkg.GoFiles, nativesPkg.TestGoFiles...)
		if isTestPkg {
			names = nativesPkg.XTestGoFiles
		}
		for _, name := range names {
			file, err := parser.ParseFile(fileSet, filepath.Join(nativesPkg.Dir, name), nil, parser.ParseComments)
			if err != nil {
				panic(err)
			}
			for _, decl := range file.Decls {
				if d, ok := decl.(*ast.FuncDecl); ok {
					replacedDeclNames[funcName(d)] = true
				}
			}
			files = append(files, file)
		}
	}
	delete(replacedDeclNames, "init")

	var errList compiler.ErrorList
	for _, name := range pkg.GoFiles {
		if !filepath.IsAbs(name) {
			name = filepath.Join(pkg.Dir, name)
		}
		r, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		file, err := parser.ParseFile(fileSet, name, r, parser.ParseComments)
		r.Close()
		if err != nil {
			if list, isList := err.(scanner.ErrorList); isList {
				for _, entry := range list {
					errList = append(errList, entry)
				}
				continue
			}
			errList = append(errList, err)
			continue
		}

		for _, decl := range file.Decls {
			if d, ok := decl.(*ast.FuncDecl); ok && replacedDeclNames[funcName(d)] {
				d.Name = ast.NewIdent("_")
			}
		}
		files = append(files, applyPatches(file, fileSet, pkg.ImportPath))
	}
	if errList != nil {
		return nil, errList
	}
	return files, nil
}

type Options struct {
	GOROOT        string
	GOPATH        string
	Verbose       bool
	Watch         bool
	CreateMapFile bool
	Minify        bool
}

type PackageData struct {
	*build.Package
	SrcModTime time.Time
	UpToDate   bool
	Archive    *compiler.Archive
}

type Session struct {
	options       *Options
	Packages      map[string]*PackageData
	ImportContext *compiler.ImportContext
	Watcher       *fsnotify.Watcher
}

func NewSession(options *Options) *Session {
	if options.GOROOT == "" {
		options.GOROOT = build.Default.GOROOT
	}
	if options.GOPATH == "" {
		options.GOPATH = build.Default.GOPATH
	}
	options.Verbose = options.Verbose || options.Watch

	s := &Session{
		options:  options,
		Packages: make(map[string]*PackageData),
	}
	s.ImportContext = compiler.NewImportContext(s.ImportPackage)
	if options.Watch {
		var err error
		s.Watcher, err = fsnotify.NewWatcher()
		if err != nil {
			panic(err)
		}
	}
	return s
}

func (s *Session) ArchSuffix() string {
	if s.options.Minify {
		return "js-min"
	}
	return "js"
}

func (s *Session) BuildDir(packagePath string, importPath string, pkgObj string) error {
	if s.Watcher != nil {
		s.Watcher.Watch(packagePath)
	}
	buildPkg, err := NewBuildContext(s.ArchSuffix()).ImportDir(packagePath, 0)
	if err != nil {
		return err
	}
	pkg := &PackageData{Package: buildPkg}
	pkg.ImportPath = importPath
	if err := s.BuildPackage(pkg); err != nil {
		return err
	}
	if pkgObj == "" {
		pkgObj = filepath.Base(packagePath) + ".js"
	}
	if err := s.WriteCommandPackage(pkg, pkgObj); err != nil {
		return err
	}
	return nil
}

func (s *Session) BuildFiles(filenames []string, pkgObj string, packagePath string) error {
	pkg := &PackageData{
		Package: &build.Package{
			Name:       "main",
			ImportPath: "main",
			Dir:        packagePath,
			GoFiles:    filenames,
		},
	}

	if err := s.BuildPackage(pkg); err != nil {
		return err
	}
	return s.WriteCommandPackage(pkg, pkgObj)
}

func (s *Session) ImportPackage(path string) (*compiler.Archive, error) {
	if pkg, found := s.Packages[path]; found {
		return pkg.Archive, nil
	}

	buildPkg, err := Import(path, build.AllowBinary, s.ArchSuffix())
	if s.Watcher != nil && buildPkg != nil { // add watch even on error
		if err := s.Watcher.Watch(buildPkg.Dir); err != nil {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}
	pkg := &PackageData{Package: buildPkg}
	if err := s.BuildPackage(pkg); err != nil {
		return nil, err
	}
	return pkg.Archive, nil
}

func (s *Session) ImportDependencies(archive *compiler.Archive) ([]*compiler.Archive, error) {
	var deps []*compiler.Archive
	for _, depPath := range archive.Dependencies {
		dep, err := s.ImportPackage(string(depPath))
		if err != nil {
			return nil, err
		}
		deps = append(deps, dep)
	}
	deps = append(deps, archive)
	return deps, nil
}

func (s *Session) BuildPackage(pkg *PackageData) error {
	s.Packages[pkg.ImportPath] = pkg
	if pkg.ImportPath == "unsafe" {
		return nil
	}

	if pkg.PkgObj != "" {
		var fileInfo os.FileInfo
		gopherjsBinary, err := osext.Executable()
		if err == nil {
			fileInfo, err = os.Stat(gopherjsBinary)
			if err == nil {
				pkg.SrcModTime = fileInfo.ModTime()
			}
		}
		if err != nil {
			os.Stderr.WriteString("Could not get GopherJS binary's modification timestamp. Please report issue.\n")
			pkg.SrcModTime = time.Now()
		}

		for _, importedPkgPath := range pkg.Imports {
			ignored := true
			for _, pos := range pkg.ImportPos[importedPkgPath] {
				importFile := filepath.Base(pos.Filename)
				for _, file := range pkg.GoFiles {
					if importFile == file {
						ignored = false
						break
					}
				}
				if !ignored {
					break
				}
			}
			if importedPkgPath == "unsafe" || ignored {
				continue
			}
			_, err := s.ImportPackage(importedPkgPath)
			if err != nil {
				return err
			}
			impModeTime := s.Packages[importedPkgPath].SrcModTime
			if impModeTime.After(pkg.SrcModTime) {
				pkg.SrcModTime = impModeTime
			}
		}

		for _, name := range pkg.GoFiles {
			fileInfo, err := os.Stat(filepath.Join(pkg.Dir, name))
			if err != nil {
				return err
			}
			if fileInfo.ModTime().After(pkg.SrcModTime) {
				pkg.SrcModTime = fileInfo.ModTime()
			}
		}

		pkgObjFileInfo, err := os.Stat(pkg.PkgObj)
		if err == nil && !pkg.SrcModTime.After(pkgObjFileInfo.ModTime()) {
			// package object is up to date, load from disk if library
			pkg.UpToDate = true
			if pkg.IsCommand() {
				return nil
			}

			objFile, err := ioutil.ReadFile(pkg.PkgObj)
			if err != nil {
				return err
			}

			pkg.Archive, err = compiler.UnmarshalArchive(pkg.PkgObj, pkg.ImportPath, objFile, s.ImportContext)
			if err != nil {
				return err
			}

			return nil
		}
	}

	fileSet := token.NewFileSet()
	files, err := Parse(pkg.Package, fileSet)
	if err != nil {
		return err
	}
	pkg.Archive, err = compiler.Compile(pkg.ImportPath, files, fileSet, s.ImportContext, s.options.Minify)
	if err != nil {
		return err
	}

	if s.options.Verbose {
		fmt.Println(pkg.ImportPath)
	}

	if pkg.PkgObj == "" || pkg.IsCommand() {
		return nil
	}

	if err := s.writeLibraryPackage(pkg, pkg.PkgObj); err != nil {
		if strings.HasPrefix(pkg.PkgObj, s.options.GOROOT) {
			// fall back to first GOPATH workspace
			firstGopathWorkspace := filepath.SplitList(s.options.GOPATH)[0]
			if err := s.writeLibraryPackage(pkg, filepath.Join(firstGopathWorkspace, pkg.PkgObj[len(s.options.GOROOT):])); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	return nil
}

func (s *Session) writeLibraryPackage(pkg *PackageData, pkgObj string) error {
	if err := os.MkdirAll(filepath.Dir(pkgObj), 0777); err != nil {
		return err
	}

	data, err := compiler.MarshalArchive(pkg.Archive)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(pkgObj, data, 0666)
}

func (s *Session) WriteCommandPackage(pkg *PackageData, pkgObj string) error {
	if !pkg.IsCommand() || pkg.UpToDate {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(pkgObj), 0777); err != nil {
		return err
	}
	codeFile, err := os.Create(pkgObj)
	if err != nil {
		return err
	}
	defer codeFile.Close()

	sourceMapFilter := &compiler.SourceMapFilter{Writer: codeFile}
	if s.options.CreateMapFile {
		m := sourcemap.Map{File: filepath.Base(pkgObj)}
		mapFile, err := os.Create(pkgObj + ".map")
		if err != nil {
			return err
		}

		defer func() {
			m.WriteTo(mapFile)
			mapFile.Close()
			fmt.Fprintf(codeFile, "//# sourceMappingURL=%s.map\n", filepath.Base(pkgObj))
		}()

		sourceMapFilter.MappingCallback = func(generatedLine, generatedColumn int, fileSet *token.FileSet, originalPos token.Pos) {
			if !originalPos.IsValid() {
				m.AddMapping(&sourcemap.Mapping{GeneratedLine: generatedLine, GeneratedColumn: generatedColumn})
				return
			}
			pos := fileSet.Position(originalPos)
			file := pos.Filename
			switch hasGopathPrefix, prefixLen := hasGopathPrefix(file, s.options.GOPATH); {
			case hasGopathPrefix:
				file = filepath.ToSlash(filepath.Join("/gopath", file[prefixLen:]))
			case strings.HasPrefix(file, s.options.GOROOT):
				file = filepath.ToSlash(filepath.Join("/goroot", file[len(s.options.GOROOT):]))
			default:
				file = filepath.Base(file)
			}
			m.AddMapping(&sourcemap.Mapping{GeneratedLine: generatedLine, GeneratedColumn: generatedColumn, OriginalFile: file, OriginalLine: pos.Line, OriginalColumn: pos.Column})
		}
	}

	deps, err := s.ImportDependencies(pkg.Archive)
	if err != nil {
		return err
	}
	return compiler.WriteProgramCode(deps, s.ImportContext, sourceMapFilter)
}

// hasGopathPrefix returns true and the length of the matched GOPATH workspace,
// iff file has a prefix that matches one of the GOPATH workspaces.
func hasGopathPrefix(file, gopath string) (hasGopathPrefix bool, prefixLen int) {
	gopathWorkspaces := filepath.SplitList(gopath)
	for _, gopathWorkspace := range gopathWorkspaces {
		if strings.HasPrefix(file, gopathWorkspace) {
			return true, len(gopathWorkspace)
		}
	}
	return false, 0
}

func (s *Session) WaitForChange() {
	fmt.Println("\x1B[32mwatching for changes...\x1B[39m")
	select {
	case ev := <-s.Watcher.Event:
		fmt.Println("\x1B[32mchange detected: " + ev.Name + "\x1B[39m")
	case err := <-s.Watcher.Error:
		fmt.Println("\x1B[32mwatcher error: " + err.Error() + "\x1B[39m")
	}
	s.Watcher.Close()
}

func applyPatches(file *ast.File, fileSet *token.FileSet, importPath string) *ast.File {
	removeImport := func(path string) {
		for _, decl := range file.Decls {
			if d, ok := decl.(*ast.GenDecl); ok && d.Tok == token.IMPORT {
				for j, spec := range d.Specs {
					value := spec.(*ast.ImportSpec).Path.Value
					if value[1:len(value)-1] == path {
						d.Specs[j] = d.Specs[len(d.Specs)-1]
						d.Specs = d.Specs[:len(d.Specs)-1]
						return
					}
				}
			}
		}
	}

	removeFunction := func(name string) {
		for i, decl := range file.Decls {
			if d, ok := decl.(*ast.FuncDecl); ok {
				if d.Name.String() == name {
					file.Decls[i] = file.Decls[len(file.Decls)-1]
					file.Decls = file.Decls[:len(file.Decls)-1]
					return
				}
			}
		}
	}

	removeType := func(name string) {
		for i, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Tok == token.TYPE {
					for j, s := range d.Specs {
						if s.(*ast.TypeSpec).Name.String() == name {
							d.Specs[j] = d.Specs[len(d.Specs)-1]
							d.Specs = d.Specs[:len(d.Specs)-1]
							break
						}
					}
				}
			case *ast.FuncDecl:
				if d.Recv != nil {
					recv := d.Recv.List[0].Type
					if s, ok := recv.(*ast.StarExpr); ok {
						recv = s.X
					}
					if recv.(*ast.Ident).Name == name {
						file.Decls[i] = file.Decls[len(file.Decls)-1]
						file.Decls = file.Decls[:len(file.Decls)-1]
					}
				}
			}
		}
	}

	basename := filepath.Base(fileSet.Position(file.Pos()).Filename)
	switch {
	case importPath == "bytes_test" && basename == "equal_test.go":
		file, _ = parser.ParseFile(fileSet, basename, "package bytes_test", 0)

	case importPath == "crypto/rc4" && basename == "rc4_ref.go": // see https://codereview.appspot.com/40540049/
		file, _ = parser.ParseFile(fileSet, basename, "// Copyright 2013 The Go Authors. All rights reserved.\n// Use of this source code is governed by a BSD-style\n// license that can be found in the LICENSE file.\n\n// +build !amd64,!arm,!386\n\npackage rc4\n\n// XORKeyStream sets dst to the result of XORing src with the key stream.\n// Dst and src may be the same slice but otherwise should not overlap.\nfunc (c *Cipher) XORKeyStream(dst, src []byte) {\n  i, j := c.i, c.j\n  for k, v := range src {\n    i += 1\n    j += uint8(c.s[i])\n    c.s[i], c.s[j] = c.s[j], c.s[i]\n    dst[k] = v ^ uint8(c.s[uint8(c.s[i]+c.s[j])])\n  }\n  c.i, c.j = i, j\n}", 0)

	case importPath == "encoding/json" && basename == "stream_test.go":
		removeImport("net")
		removeFunction("TestBlocking")

	case importPath == "math/big" && basename == "arith.go": // see https://codereview.appspot.com/55470046/
		file, _ = parser.ParseFile(fileSet, basename, "// Copyright 2009 The Go Authors. All rights reserved.\n// Use of this source code is governed by a BSD-style\n// license that can be found in the LICENSE file.\n\n// This file provides Go implementations of elementary multi-precision\n// arithmetic operations on word vectors. Needed for platforms without\n// assembly implementations of these routines.\n\npackage big\n\n// A Word represents a single digit of a multi-precision unsigned integer.\ntype Word uintptr\n\nconst (\n	// Compute the size _S of a Word in bytes.\n	_m    = ^Word(0)\n	_logS = _m>>8&1 + _m>>16&1 + _m>>32&1\n	_S    = 1 << _logS\n\n	_W = _S << 3 // word size in bits\n	_B = 1 << _W // digit base\n	_M = _B - 1  // digit mask\n\n	_W2 = _W / 2   // half word size in bits\n	_B2 = 1 << _W2 // half digit base\n	_M2 = _B2 - 1  // half digit mask\n)\n\n// ----------------------------------------------------------------------------\n// Elementary operations on words\n//\n// These operations are used by the vector operations below.\n\n// z1<<_W + z0 = x+y+c, with c == 0 or 1\nfunc addWW_g(x, y, c Word) (z1, z0 Word) {\n	yc := y + c\n	z0 = x + yc\n	if z0 < x || yc < y {\n		z1 = 1\n	}\n	return\n}\n\n// z1<<_W + z0 = x-y-c, with c == 0 or 1\nfunc subWW_g(x, y, c Word) (z1, z0 Word) {\n	yc := y + c\n	z0 = x - yc\n	if z0 > x || yc < y {\n		z1 = 1\n	}\n	return\n}\n\n// z1<<_W + z0 = x*y\n// Adapted from Warren, Hacker's Delight, p. 132.\nfunc mulWW_g(x, y Word) (z1, z0 Word) {\n	x0 := x & _M2\n	x1 := x >> _W2\n	y0 := y & _M2\n	y1 := y >> _W2\n	w0 := x0 * y0\n	t := x1*y0 + w0>>_W2\n	w1 := t & _M2\n	w2 := t >> _W2\n	w1 += x0 * y1\n	z1 = x1*y1 + w2 + w1>>_W2\n	z0 = x * y\n	return\n}\n\n// z1<<_W + z0 = x*y + c\nfunc mulAddWWW_g(x, y, c Word) (z1, z0 Word) {\n	z1, zz0 := mulWW(x, y)\n	if z0 = zz0 + c; z0 < zz0 {\n		z1++\n	}\n	return\n}\n\n// Length of x in bits.\nfunc bitLen_g(x Word) (n int) {\n	for ; x >= 0x8000; x >>= 16 {\n		n += 16\n	}\n	if x >= 0x80 {\n		x >>= 8\n		n += 8\n	}\n	if x >= 0x8 {\n		x >>= 4\n		n += 4\n	}\n	if x >= 0x2 {\n		x >>= 2\n		n += 2\n	}\n	if x >= 0x1 {\n		n++\n	}\n	return\n}\n\n// log2 computes the integer binary logarithm of x.\n// The result is the integer n for which 2^n <= x < 2^(n+1).\n// If x == 0, the result is -1.\nfunc log2(x Word) int {\n	return bitLen(x) - 1\n}\n\n// Number of leading zeros in x.\nfunc leadingZeros(x Word) uint {\n	return uint(_W - bitLen(x))\n}\n\n// q = (u1<<_W + u0 - r)/y\n// Adapted from Warren, Hacker's Delight, p. 152.\nfunc divWW_g(u1, u0, v Word) (q, r Word) {\n	if u1 >= v {\n		return 1<<_W - 1, 1<<_W - 1\n	}\n\n	s := leadingZeros(v)\n	v <<= s\n\n	vn1 := v >> _W2\n	vn0 := v & _M2\n	un32 := u1<<s | u0>>(_W-s)\n	un10 := u0 << s\n	un1 := un10 >> _W2\n	un0 := un10 & _M2\n	q1 := un32 / vn1\n	rhat := un32 - q1*vn1\n\n	for q1 >= _B2 || q1*vn0 > _B2*rhat+un1 {\n		q1--\n		rhat += vn1\n		if rhat >= _B2 {\n			break\n		}\n	}\n\n	un21 := un32*_B2 + un1 - q1*v\n	q0 := un21 / vn1\n	rhat = un21 - q0*vn1\n\n	for q0 >= _B2 || q0*vn0 > _B2*rhat+un0 {\n		q0--\n		rhat += vn1\n		if rhat >= _B2 {\n			break\n		}\n	}\n\n	return q1*_B2 + q0, (un21*_B2 + un0 - q0*v) >> s\n}\n\nfunc addVV_g(z, x, y []Word) (c Word) {\n	for i := range z {\n		c, z[i] = addWW_g(x[i], y[i], c)\n	}\n	return\n}\n\nfunc subVV_g(z, x, y []Word) (c Word) {\n	for i := range z {\n		c, z[i] = subWW_g(x[i], y[i], c)\n	}\n	return\n}\n\nfunc addVW_g(z, x []Word, y Word) (c Word) {\n	c = y\n	for i := range z {\n		c, z[i] = addWW_g(x[i], c, 0)\n	}\n	return\n}\n\nfunc subVW_g(z, x []Word, y Word) (c Word) {\n	c = y\n	for i := range z {\n		c, z[i] = subWW_g(x[i], c, 0)\n	}\n	return\n}\n\nfunc shlVU_g(z, x []Word, s uint) (c Word) {\n	if n := len(z); n > 0 {\n		ŝ := _W - s\n		w1 := x[n-1]\n		c = w1 >> ŝ\n		for i := n - 1; i > 0; i-- {\n			w := w1\n			w1 = x[i-1]\n			z[i] = w<<s | w1>>ŝ\n		}\n		z[0] = w1 << s\n	}\n	return\n}\n\nfunc shrVU_g(z, x []Word, s uint) (c Word) {\n	if n := len(z); n > 0 {\n		ŝ := _W - s\n		w1 := x[0]\n		c = w1 << ŝ\n		for i := 0; i < n-1; i++ {\n			w := w1\n			w1 = x[i+1]\n			z[i] = w>>s | w1<<ŝ\n		}\n		z[n-1] = w1 >> s\n	}\n	return\n}\n\nfunc mulAddVWW_g(z, x []Word, y, r Word) (c Word) {\n	c = r\n	for i := range z {\n		c, z[i] = mulAddWWW_g(x[i], y, c)\n	}\n	return\n}\n\nfunc addMulVVW_g(z, x []Word, y Word) (c Word) {\n	for i := range z {\n		z1, z0 := mulAddWWW_g(x[i], y, z[i])\n		c, z[i] = addWW_g(z0, c, 0)\n		c += z1\n	}\n	return\n}\n\nfunc divWVW_g(z []Word, xn Word, x []Word, y Word) (r Word) {\n	r = xn\n	for i := len(z) - 1; i >= 0; i-- {\n		z[i], r = divWW_g(r, x[i], y)\n	}\n	return\n}", 0)

	case importPath == "reflect_test" && basename == "all_test.go":
		removeImport("unsafe")
		removeFunction("TestAlignment")
		removeFunction("TestSliceOverflow")
		removeFunction("TestCallMethodJump")
		removeType("Inner")
		removeType("Outer")
		file.Comments = nil

	case importPath == "runtime" && strings.HasPrefix(basename, "zgoarch_"):
		file, _ = parser.ParseFile(fileSet, basename, "package runtime\nconst theGoarch = `js`\n", 0)

	case importPath == "sync" && basename == "pool.go":
		file, _ = parser.ParseFile(fileSet, basename, "package sync", 0)

	case importPath == "sync/atomic_test" && basename == "atomic_test.go":
		removeFunction("TestUnaligned64")

	case importPath == "text/template/parse" && basename == "lex.go": // this patch will be removed as soon as goroutines are supported
		file, _ = parser.ParseFile(fileSet, basename, "// Copyright 2011 The Go Authors. All rights reserved.\n// Use of this source code is governed by a BSD-style\n// license that can be found in the LICENSE file.\npackage parse\nimport (\n	\"container/list\"\n	\"fmt\"\n	\"strings\"\n	\"unicode\"\n	\"unicode/utf8\"\n)\n// item represents a token or text string returned from the scanner.\ntype item struct {\n	typ itemType // The type of this item.\n	pos Pos      // The starting position, in bytes, of this item in the input string.\n	val string   // The value of this item.\n}\nfunc (i item) String() string {\n	switch {\n	case i.typ == itemEOF:\n		return \"EOF\"\n	case i.typ == itemError:\n		return i.val\n	case i.typ > itemKeyword:\n		return fmt.Sprintf(\"<%s>\", i.val)\n	case len(i.val) > 10:\n		return fmt.Sprintf(\"%.10q...\", i.val)\n	}\n	return fmt.Sprintf(\"%q\", i.val)\n}\n// itemType identifies the type of lex items.\ntype itemType int\nconst (\n	itemError        itemType = iota // error occurred; value is text of error\n	itemBool                         // boolean constant\n	itemChar                         // printable ASCII character; grab bag for comma etc.\n	itemCharConstant                 // character constant\n	itemComplex                      // complex constant (1+2i); imaginary is just a number\n	itemColonEquals                  // colon-equals (':=') introducing a declaration\n	itemEOF\n	itemField      // alphanumeric identifier starting with '.'\n	itemIdentifier // alphanumeric identifier not starting with '.'\n	itemLeftDelim  // left action delimiter\n	itemLeftParen  // '(' inside action\n	itemNumber     // simple number, including imaginary\n	itemPipe       // pipe symbol\n	itemRawString  // raw quoted string (includes quotes)\n	itemRightDelim // right action delimiter\n	itemRightParen // ')' inside action\n	itemSpace      // run of spaces separating arguments\n	itemString     // quoted string (includes quotes)\n	itemText       // plain text\n	itemVariable   // variable starting with '$', such as '$' or  '$1' or '$hello'\n	// Keywords appear after all the rest.\n	itemKeyword  // used only to delimit the keywords\n	itemDot      // the cursor, spelled '.'\n	itemDefine   // define keyword\n	itemElse     // else keyword\n	itemEnd      // end keyword\n	itemIf       // if keyword\n	itemNil      // the untyped nil constant, easiest to treat as a keyword\n	itemRange    // range keyword\n	itemTemplate // template keyword\n	itemWith     // with keyword\n)\nvar key = map[string]itemType{\n	\".\":        itemDot,\n	\"define\":   itemDefine,\n	\"else\":     itemElse,\n	\"end\":      itemEnd,\n	\"if\":       itemIf,\n	\"range\":    itemRange,\n	\"nil\":      itemNil,\n	\"template\": itemTemplate,\n	\"with\":     itemWith,\n}\nconst eof = -1\n// stateFn represents the state of the scanner as a function that returns the next state.\ntype stateFn func(*lexer) stateFn\n// lexer holds the state of the scanner.\ntype lexer struct {\n	name       string     // the name of the input; used only for error reports\n	input      string     // the string being scanned\n	leftDelim  string     // start of action\n	rightDelim string     // end of action\n	state      stateFn    // the next lexing function to enter\n	pos        Pos        // current position in the input\n	start      Pos        // start position of this item\n	width      Pos        // width of last rune read from input\n	lastPos    Pos        // position of most recent item returned by nextItem\n	items      *list.List // scanned items\n	parenDepth int        // nesting depth of ( ) exprs\n}\n// next returns the next rune in the input.\nfunc (l *lexer) next() rune {\n	if int(l.pos) >= len(l.input) {\n		l.width = 0\n		return eof\n	}\n	r, w := utf8.DecodeRuneInString(l.input[l.pos:])\n	l.width = Pos(w)\n	l.pos += l.width\n	return r\n}\n// peek returns but does not consume the next rune in the input.\nfunc (l *lexer) peek() rune {\n	r := l.next()\n	l.backup()\n	return r\n}\n// backup steps back one rune. Can only be called once per call of next.\nfunc (l *lexer) backup() {\n	l.pos -= l.width\n}\n// emit passes an item back to the client.\nfunc (l *lexer) emit(t itemType) {\n	l.items.PushBack(item{t, l.start, l.input[l.start:l.pos]})\n	l.start = l.pos\n}\n// ignore skips over the pending input before this point.\nfunc (l *lexer) ignore() {\n	l.start = l.pos\n}\n// accept consumes the next rune if it's from the valid set.\nfunc (l *lexer) accept(valid string) bool {\n	if strings.IndexRune(valid, l.next()) >= 0 {\n		return true\n	}\n	l.backup()\n	return false\n}\n// acceptRun consumes a run of runes from the valid set.\nfunc (l *lexer) acceptRun(valid string) {\n	for strings.IndexRune(valid, l.next()) >= 0 {\n	}\n	l.backup()\n}\n// lineNumber reports which line we're on, based on the position of\n// the previous item returned by nextItem. Doing it this way\n// means we don't have to worry about peek double counting.\nfunc (l *lexer) lineNumber() int {\n	return 1 + strings.Count(l.input[:l.lastPos], \"\\n\")\n}\n// errorf returns an error token and terminates the scan by passing\n// back a nil pointer that will be the next state, terminating l.nextItem.\nfunc (l *lexer) errorf(format string, args ...interface{}) stateFn {\n	l.items.PushBack(item{itemError, l.start, fmt.Sprintf(format, args...)})\n	return nil\n}\n// nextItem returns the next item from the input.\nfunc (l *lexer) nextItem() item {\n	element := l.items.Front()\n	for element == nil {\n		l.state = l.state(l)\n		element = l.items.Front()\n	}\n	l.items.Remove(element)\n	item := element.Value.(item)\n	l.lastPos = item.pos\n	return item\n}\n// lex creates a new scanner for the input string.\nfunc lex(name, input, left, right string) *lexer {\n	if left == \"\" {\n		left = leftDelim\n	}\n	if right == \"\" {\n		right = rightDelim\n	}\n	l := &lexer{\n		name:       name,\n		input:      input,\n		leftDelim:  left,\n		rightDelim: right,\n		items:      list.New(),\n	}\n	l.state = lexText\n	return l\n}\n// state functions\nconst (\n	leftDelim    = \"{{\"\n	rightDelim   = \"}}\"\n	leftComment  = \"/*\"\n	rightComment = \"*/\"\n)\n// lexText scans until an opening action delimiter, \"{{\".\nfunc lexText(l *lexer) stateFn {\n	for {\n		if strings.HasPrefix(l.input[l.pos:], l.leftDelim) {\n			if l.pos > l.start {\n				l.emit(itemText)\n			}\n			return lexLeftDelim\n		}\n		if l.next() == eof {\n			break\n		}\n	}\n	// Correctly reached EOF.\n	if l.pos > l.start {\n		l.emit(itemText)\n	}\n	l.emit(itemEOF)\n	return nil\n}\n// lexLeftDelim scans the left delimiter, which is known to be present.\nfunc lexLeftDelim(l *lexer) stateFn {\n	l.pos += Pos(len(l.leftDelim))\n	if strings.HasPrefix(l.input[l.pos:], leftComment) {\n		return lexComment\n	}\n	l.emit(itemLeftDelim)\n	l.parenDepth = 0\n	return lexInsideAction\n}\n// lexComment scans a comment. The left comment marker is known to be present.\nfunc lexComment(l *lexer) stateFn {\n	l.pos += Pos(len(leftComment))\n	i := strings.Index(l.input[l.pos:], rightComment)\n	if i < 0 {\n		return l.errorf(\"unclosed comment\")\n	}\n	l.pos += Pos(i + len(rightComment))\n	if !strings.HasPrefix(l.input[l.pos:], l.rightDelim) {\n		return l.errorf(\"comment ends before closing delimiter\")\n	}\n	l.pos += Pos(len(l.rightDelim))\n	l.ignore()\n	return lexText\n}\n// lexRightDelim scans the right delimiter, which is known to be present.\nfunc lexRightDelim(l *lexer) stateFn {\n	l.pos += Pos(len(l.rightDelim))\n	l.emit(itemRightDelim)\n	return lexText\n}\n// lexInsideAction scans the elements inside action delimiters.\nfunc lexInsideAction(l *lexer) stateFn {\n	// Either number, quoted string, or identifier.\n	// Spaces separate arguments; runs of spaces turn into itemSpace.\n	// Pipe symbols separate and are emitted.\n	if strings.HasPrefix(l.input[l.pos:], l.rightDelim) {\n		if l.parenDepth == 0 {\n			return lexRightDelim\n		}\n		return l.errorf(\"unclosed left paren\")\n	}\n	switch r := l.next(); {\n	case r == eof || isEndOfLine(r):\n		return l.errorf(\"unclosed action\")\n	case isSpace(r):\n		return lexSpace\n	case r == ':':\n		if l.next() != '=' {\n			return l.errorf(\"expected :=\")\n		}\n		l.emit(itemColonEquals)\n	case r == '|':\n		l.emit(itemPipe)\n	case r == '\"':\n		return lexQuote\n	case r == '`':\n		return lexRawQuote\n	case r == '$':\n		return lexVariable\n	case r == '\\'':\n		return lexChar\n	case r == '.':\n		// special look-ahead for \".field\" so we don't break l.backup().\n		if l.pos < Pos(len(l.input)) {\n			r := l.input[l.pos]\n			if r < '0' || '9' < r {\n				return lexField\n			}\n		}\n		fallthrough // '.' can start a number.\n	case r == '+' || r == '-' || ('0' <= r && r <= '9'):\n		l.backup()\n		return lexNumber\n	case isAlphaNumeric(r):\n		l.backup()\n		return lexIdentifier\n	case r == '(':\n		l.emit(itemLeftParen)\n		l.parenDepth++\n		return lexInsideAction\n	case r == ')':\n		l.emit(itemRightParen)\n		l.parenDepth--\n		if l.parenDepth < 0 {\n			return l.errorf(\"unexpected right paren %#U\", r)\n		}\n		return lexInsideAction\n	case r <= unicode.MaxASCII && unicode.IsPrint(r):\n		l.emit(itemChar)\n		return lexInsideAction\n	default:\n		return l.errorf(\"unrecognized character in action: %#U\", r)\n	}\n	return lexInsideAction\n}\n// lexSpace scans a run of space characters.\n// One space has already been seen.\nfunc lexSpace(l *lexer) stateFn {\n	for isSpace(l.peek()) {\n		l.next()\n	}\n	l.emit(itemSpace)\n	return lexInsideAction\n}\n// lexIdentifier scans an alphanumeric.\nfunc lexIdentifier(l *lexer) stateFn {\nLoop:\n	for {\n		switch r := l.next(); {\n		case isAlphaNumeric(r):\n			// absorb.\n		default:\n			l.backup()\n			word := l.input[l.start:l.pos]\n			if !l.atTerminator() {\n				return l.errorf(\"bad character %#U\", r)\n			}\n			switch {\n			case key[word] > itemKeyword:\n				l.emit(key[word])\n			case word[0] == '.':\n				l.emit(itemField)\n			case word == \"true\", word == \"false\":\n				l.emit(itemBool)\n			default:\n				l.emit(itemIdentifier)\n			}\n			break Loop\n		}\n	}\n	return lexInsideAction\n}\n// lexField scans a field: .Alphanumeric.\n// The . has been scanned.\nfunc lexField(l *lexer) stateFn {\n	return lexFieldOrVariable(l, itemField)\n}\n// lexVariable scans a Variable: $Alphanumeric.\n// The $ has been scanned.\nfunc lexVariable(l *lexer) stateFn {\n	if l.atTerminator() { // Nothing interesting follows -> \"$\".\n		l.emit(itemVariable)\n		return lexInsideAction\n	}\n	return lexFieldOrVariable(l, itemVariable)\n}\n// lexVariable scans a field or variable: [.$]Alphanumeric.\n// The . or $ has been scanned.\nfunc lexFieldOrVariable(l *lexer, typ itemType) stateFn {\n	if l.atTerminator() { // Nothing interesting follows -> \".\" or \"$\".\n		if typ == itemVariable {\n			l.emit(itemVariable)\n		} else {\n			l.emit(itemDot)\n		}\n		return lexInsideAction\n	}\n	var r rune\n	for {\n		r = l.next()\n		if !isAlphaNumeric(r) {\n			l.backup()\n			break\n		}\n	}\n	if !l.atTerminator() {\n		return l.errorf(\"bad character %#U\", r)\n	}\n	l.emit(typ)\n	return lexInsideAction\n}\n// atTerminator reports whether the input is at valid termination character to\n// appear after an identifier. Breaks .X.Y into two pieces. Also catches cases\n// like \"$x+2\" not being acceptable without a space, in case we decide one\n// day to implement arithmetic.\nfunc (l *lexer) atTerminator() bool {\n	r := l.peek()\n	if isSpace(r) || isEndOfLine(r) {\n		return true\n	}\n	switch r {\n	case eof, '.', ',', '|', ':', ')', '(':\n		return true\n	}\n	// Does r start the delimiter? This can be ambiguous (with delim==\"//\", $x/2 will\n	// succeed but should fail) but only in extremely rare cases caused by willfully\n	// bad choice of delimiter.\n	if rd, _ := utf8.DecodeRuneInString(l.rightDelim); rd == r {\n		return true\n	}\n	return false\n}\n// lexChar scans a character constant. The initial quote is already\n// scanned. Syntax checking is done by the parser.\nfunc lexChar(l *lexer) stateFn {\nLoop:\n	for {\n		switch l.next() {\n		case '\\\\':\n			if r := l.next(); r != eof && r != '\\n' {\n				break\n			}\n			fallthrough\n		case eof, '\\n':\n			return l.errorf(\"unterminated character constant\")\n		case '\\'':\n			break Loop\n		}\n	}\n	l.emit(itemCharConstant)\n	return lexInsideAction\n}\n// lexNumber scans a number: decimal, octal, hex, float, or imaginary. This\n// isn't a perfect number scanner - for instance it accepts \".\" and \"0x0.2\"\n// and \"089\" - but when it's wrong the input is invalid and the parser (via\n// strconv) will notice.\nfunc lexNumber(l *lexer) stateFn {\n	if !l.scanNumber() {\n		return l.errorf(\"bad number syntax: %q\", l.input[l.start:l.pos])\n	}\n	if sign := l.peek(); sign == '+' || sign == '-' {\n		// Complex: 1+2i. No spaces, must end in 'i'.\n		if !l.scanNumber() || l.input[l.pos-1] != 'i' {\n			return l.errorf(\"bad number syntax: %q\", l.input[l.start:l.pos])\n		}\n		l.emit(itemComplex)\n	} else {\n		l.emit(itemNumber)\n	}\n	return lexInsideAction\n}\nfunc (l *lexer) scanNumber() bool {\n	// Optional leading sign.\n	l.accept(\"+-\")\n	// Is it hex?\n	digits := \"0123456789\"\n	if l.accept(\"0\") && l.accept(\"xX\") {\n		digits = \"0123456789abcdefABCDEF\"\n	}\n	l.acceptRun(digits)\n	if l.accept(\".\") {\n		l.acceptRun(digits)\n	}\n	if l.accept(\"eE\") {\n		l.accept(\"+-\")\n		l.acceptRun(\"0123456789\")\n	}\n	// Is it imaginary?\n	l.accept(\"i\")\n	// Next thing mustn't be alphanumeric.\n	if isAlphaNumeric(l.peek()) {\n		l.next()\n		return false\n	}\n	return true\n}\n// lexQuote scans a quoted string.\nfunc lexQuote(l *lexer) stateFn {\nLoop:\n	for {\n		switch l.next() {\n		case '\\\\':\n			if r := l.next(); r != eof && r != '\\n' {\n				break\n			}\n			fallthrough\n		case eof, '\\n':\n			return l.errorf(\"unterminated quoted string\")\n		case '\"':\n			break Loop\n		}\n	}\n	l.emit(itemString)\n	return lexInsideAction\n}\n// lexRawQuote scans a raw quoted string.\nfunc lexRawQuote(l *lexer) stateFn {\nLoop:\n	for {\n		switch l.next() {\n		case eof, '\\n':\n			return l.errorf(\"unterminated raw quoted string\")\n		case '`':\n			break Loop\n		}\n	}\n	l.emit(itemRawString)\n	return lexInsideAction\n}\n// isSpace reports whether r is a space character.\nfunc isSpace(r rune) bool {\n	return r == ' ' || r == '\\t'\n}\n// isEndOfLine reports whether r is an end-of-line character.\nfunc isEndOfLine(r rune) bool {\n	return r == '\\r' || r == '\\n'\n}\n// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.\nfunc isAlphaNumeric(r rune) bool {\n	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)\n}", 0)
	}

	return file
}
