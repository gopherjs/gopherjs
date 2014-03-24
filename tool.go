package main

import (
	"bitbucket.org/kardianos/osext"
	"code.google.com/p/go.tools/go/types"
	"flag"
	"fmt"
	"github.com/gopherjs/gopherjs/translator"
	"github.com/neelance/sourcemap"
	"go/ast"
	"go/build"
	"go/parser"
	"go/scanner"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type packageData struct {
	*build.Package
	SrcModTime time.Time
	UpToDate   bool
	Archive    *translator.Archive
}

type importCError struct{}

func (e *importCError) Error() string {
	return `importing "C" is not supported by GopherJS`
}

var currentDirectory string

func init() {
	var err error
	currentDirectory, err = os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	currentDirectory, err = filepath.EvalSymlinks(currentDirectory)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func buildImport(path string, mode build.ImportMode) (*build.Package, error) {
	if path == "C" {
		return nil, &importCError{}
	}

	buildContext := &build.Context{
		GOROOT:   build.Default.GOROOT,
		GOPATH:   build.Default.GOPATH,
		GOOS:     build.Default.GOOS,
		GOARCH:   "js",
		Compiler: "gc",
	}
	if path == "runtime" || path == "syscall" {
		buildContext.GOARCH = build.Default.GOARCH
		buildContext.InstallSuffix = "js"
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
		gopathPkgObj := build.Default.GOPATH + pkg.PkgObj[len(build.Default.GOROOT):]
		if _, err := os.Stat(gopathPkgObj); err == nil {
			pkg.PkgObj = gopathPkgObj
		}
	}
	return pkg, err
}

type session struct {
	t              *translator.Translator
	verboseInstall bool
	packages       map[string]*packageData
}

func NewSession(verboseInstall bool) *session {
	return &session{
		t:              translator.New(),
		verboseInstall: verboseInstall,
		packages:       make(map[string]*packageData),
	}
}

func main() {
	switch err := tool().(type) {
	case nil:
		os.Exit(0)
	case translator.ErrorList:
		makeRel := func(name string) string {
			if relname, err := filepath.Rel(currentDirectory, name); err == nil {
				if relname[0] != '.' {
					return "." + string(filepath.Separator) + relname
				}
				return relname
			}
			return name
		}
		for _, entry := range err {
			switch e := entry.(type) {
			case *scanner.Error:
				fmt.Fprintf(os.Stderr, "%s:%d:%d: %s\n", makeRel(e.Pos.Filename), e.Pos.Line, e.Pos.Column, e.Msg)
			case types.Error:
				pos := e.Fset.Position(e.Pos)
				fmt.Fprintf(os.Stderr, "%s:%d:%d: %s\n", makeRel(pos.Filename), pos.Line, pos.Column, e.Msg)
			default:
				fmt.Fprintln(os.Stderr, entry)
			}
		}
		os.Exit(1)
	case *exec.ExitError:
		os.Exit(err.Sys().(syscall.WaitStatus).ExitStatus())
	default:
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func tool() error {
	flag.Parse()

	cmd := flag.Arg(0)
	switch cmd {
	case "build":
		buildFlags := flag.NewFlagSet("build", flag.ContinueOnError)
		var pkgObj string
		buildFlags.StringVar(&pkgObj, "o", "", "")
		buildFlags.Parse(flag.Args()[1:])

		s := NewSession(false)

		if buildFlags.NArg() == 0 {
			buildContext := &build.Context{
				GOROOT:   build.Default.GOROOT,
				GOPATH:   build.Default.GOPATH,
				GOOS:     build.Default.GOOS,
				GOARCH:   "js",
				Compiler: "gc",
			}
			buildPkg, err := buildContext.ImportDir(currentDirectory, 0)
			if err != nil {
				return err
			}
			pkg := &packageData{Package: buildPkg}
			pkg.ImportPath = currentDirectory
			if err := s.buildPackage(pkg); err != nil {
				return err
			}
			if pkgObj == "" {
				pkgObj = filepath.Base(currentDirectory) + ".js"
			}
			if err := s.writeCommandPackage(pkg, pkgObj); err != nil {
				return err
			}
			return nil
		}

		if strings.HasSuffix(buildFlags.Arg(0), ".go") {
			for _, arg := range buildFlags.Args() {
				if !strings.HasSuffix(arg, ".go") {
					return fmt.Errorf("named files must be .go files")
				}
			}
			if pkgObj == "" {
				basename := filepath.Base(buildFlags.Arg(0))
				pkgObj = basename[:len(basename)-3] + ".js"
			}
			if err := s.buildFiles(buildFlags.Args(), pkgObj); err != nil {
				return err
			}
			return nil
		}

		for _, pkgPath := range buildFlags.Args() {
			buildPkg, err := buildImport(filepath.ToSlash(pkgPath), 0)
			if err != nil {
				return err
			}
			pkg := &packageData{Package: buildPkg}
			if err := s.buildPackage(pkg); err != nil {
				return err
			}
			if pkgObj == "" {
				pkgObj = filepath.Base(buildFlags.Arg(0)) + ".js"
			}
			if err := s.writeCommandPackage(pkg, pkgObj); err != nil {
				return err
			}
		}
		return nil

	case "install":
		installFlags := flag.NewFlagSet("install", flag.ContinueOnError)
		verbose := installFlags.Bool("v", false, "verbose")
		installFlags.Parse(flag.Args()[1:])

		s := NewSession(*verbose)

		pkgs := installFlags.Args()
		if len(pkgs) == 0 {
			srcDir, err := filepath.EvalSymlinks(filepath.Join(build.Default.GOPATH, "src"))
			if err != nil {
				return err
			}
			if !strings.HasPrefix(currentDirectory, srcDir) {
				return fmt.Errorf("gopherjs install: no install location for directory %s outside GOPATH", currentDirectory)
			}
			pkgPath, err := filepath.Rel(srcDir, currentDirectory)
			if err != nil {
				return err
			}
			pkgs = []string{pkgPath}
		}
		for _, pkgPath := range pkgs {
			pkgPath = filepath.ToSlash(pkgPath)
			if _, err := s.importPackage(pkgPath); err != nil {
				return err
			}
			pkg := s.packages[pkgPath]
			if err := s.writeCommandPackage(pkg, pkg.PkgObj); err != nil {
				return err
			}
		}
		return nil

	case "run":
		lastSourceArg := 1
		for {
			if !strings.HasSuffix(flag.Arg(lastSourceArg), ".go") {
				break
			}
			lastSourceArg++
		}
		if lastSourceArg == 1 {
			return fmt.Errorf("gopherjs run: no go files listed")
		}

		tempfile, err := ioutil.TempFile("", filepath.Base(flag.Arg(1))+".")
		if err != nil {
			return err
		}
		defer func() {
			tempfile.Close()
			os.Remove(tempfile.Name())
		}()

		s := NewSession(false)
		if err := s.buildFiles(flag.Args()[1:lastSourceArg], tempfile.Name()); err != nil {
			return err
		}
		if err := runNode(tempfile.Name(), flag.Args()[lastSourceArg:], ""); err != nil {
			return err
		}
		return nil

	case "test":
		testFlags := flag.NewFlagSet("test", flag.ContinueOnError)
		verbose := testFlags.Bool("v", false, "verbose")
		short := testFlags.Bool("short", false, "short")
		testFlags.Parse(flag.Args()[1:])

		var exitErr error
		for _, pkgPath := range testFlags.Args() {
			pkgPath = filepath.ToSlash(pkgPath)

			s := NewSession(false)

			buildPkg, err := buildImport(pkgPath, 0)
			if err != nil {
				return err
			}
			buildPkg.PkgObj = ""
			buildPkg.GoFiles = append(buildPkg.GoFiles, buildPkg.TestGoFiles...)
			pkg := &packageData{Package: buildPkg}
			if err := s.buildPackage(pkg); err != nil {
				return err
			}

			mainPkg := &packageData{
				Package: &build.Package{
					Name:       "main",
					ImportPath: "main",
				},
				Archive: &translator.Archive{
					ImportPath: "main",
				},
			}
			s.packages["main"] = mainPkg
			s.t.NewEmptyTypesPackage("main")
			testingOutput, _ := s.importPackage("testing")
			mainPkg.Archive.AddDependenciesOf(testingOutput)

			var mainFunc translator.Decl
			var names []string
			var tests []string
			collectTests := func(pkg *packageData) {
				for _, name := range pkg.Archive.Tests {
					names = append(names, name)
					tests = append(tests, fmt.Sprintf(`go$packages["%s"].%s`, pkg.ImportPath, name))
					mainFunc.DceDeps = append(mainFunc.DceDeps, translator.DepId(pkg.ImportPath+":"+name))
				}
				mainPkg.Archive.AddDependenciesOf(pkg.Archive)
			}

			collectTests(pkg)
			if len(pkg.XTestGoFiles) != 0 {
				testPkg := &packageData{Package: &build.Package{
					ImportPath: pkg.ImportPath + "_test",
					Dir:        pkg.Dir,
					GoFiles:    pkg.XTestGoFiles,
				}}
				if err := s.buildPackage(testPkg); err != nil {
					return err
				}
				collectTests(testPkg)
			}

			mainFunc.DceDeps = append(mainFunc.DceDeps, translator.DepId("flag:Parse"))
			mainFunc.BodyCode = []byte(fmt.Sprintf(`
				go$pkg.main = function() {
					go$packages["testing"].Main2("%s", "%s", ["%s"], [%s]);
				};
			`, pkg.ImportPath, pkg.Dir, strings.Join(names, `", "`), strings.Join(tests, ", ")))

			mainPkg.Archive.Declarations = []translator.Decl{mainFunc}
			mainPkg.Archive.AddDependency("main")

			tempfile, err := ioutil.TempFile("", "test.")
			if err != nil {
				return err
			}
			defer func() {
				tempfile.Close()
				os.Remove(tempfile.Name())
			}()

			if err := s.writeCommandPackage(mainPkg, tempfile.Name()); err != nil {
				return err
			}

			var args []string
			if *verbose {
				args = append(args, "-test.v")
			}
			if *short {
				args = append(args, "-test.short")
			}
			if err := runNode(tempfile.Name(), args, ""); err != nil {
				if _, ok := err.(*exec.ExitError); !ok {
					return err
				}
				exitErr = err
			}
		}
		return exitErr

	case "tool":
		tool := flag.Arg(1)
		toolFlags := flag.NewFlagSet("tool", flag.ContinueOnError)
		toolFlags.Bool("e", false, "")
		toolFlags.Bool("l", false, "")
		toolFlags.Bool("m", false, "")
		toolFlags.String("o", "", "")
		toolFlags.String("D", "", "")
		toolFlags.String("I", "", "")
		toolFlags.Parse(flag.Args()[2:])

		if len(tool) == 2 {
			switch tool[1] {
			case 'g':
				basename := filepath.Base(toolFlags.Arg(0))
				s := NewSession(false)
				if err := s.buildFiles([]string{toolFlags.Arg(0)}, basename[:len(basename)-3]+".js"); err != nil {
					return err
				}
				return nil
			}
		}
		return fmt.Errorf("Tool not supported: " + tool)

	case "help", "":
		os.Stderr.WriteString(`GopherJS is a tool for compiling Go source code to JavaScript.

Usage:

    gopherjs command [arguments]

The commands are:

    build       compile packages and dependencies
    install     compile and install packages and dependencies
    run         compile and run Go program
    test        test packages

`)
		return nil

	default:
		fmt.Fprintf(os.Stderr, "gopherjs: unknown subcommand \"%s\"\nRun 'gopherjs help' for usage.\n", cmd)
		return nil

	}
}

func (s *session) buildFiles(filenames []string, pkgObj string) error {
	pkg := &packageData{
		Package: &build.Package{
			Name:       "main",
			ImportPath: "main",
			Dir:        currentDirectory,
			GoFiles:    filenames,
		},
	}

	if err := s.buildPackage(pkg); err != nil {
		return err
	}
	return s.writeCommandPackage(pkg, pkgObj)
}

func (s *session) importPackage(path string) (*translator.Archive, error) {
	if pkg, found := s.packages[path]; found {
		return pkg.Archive, nil
	}

	buildPkg, err := buildImport(path, build.AllowBinary)
	if err != nil {
		return nil, err
	}
	pkg := &packageData{Package: buildPkg}
	if err := s.buildPackage(pkg); err != nil {
		return nil, err
	}

	return pkg.Archive, nil
}

func (s *session) buildPackage(pkg *packageData) error {
	s.packages[pkg.ImportPath] = pkg
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
			if importedPkgPath == "unsafe" {
				continue
			}
			_, err := s.importPackage(importedPkgPath)
			if err != nil {
				return err
			}
			impModeTime := s.packages[importedPkgPath].SrcModTime
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

			pkg.Archive, err = s.t.ReadArchive(pkg.PkgObj, pkg.ImportPath, objFile)
			if err != nil {
				return err
			}

			return nil
		}
	}

	if s.verboseInstall {
		fmt.Println(pkg.ImportPath)
	}

	fileSet := token.NewFileSet()
	var files []*ast.File
	var errList translator.ErrorList
	for _, name := range pkg.GoFiles {
		if !filepath.IsAbs(name) {
			name = filepath.Join(pkg.Dir, name)
		}
		r, err := os.Open(name)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(fileSet, name, r, 0)
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

		files = append(files, s.applyPatches(file, fileSet, pkg.ImportPath, filepath.Base(name)))
	}
	if errList != nil {
		return errList
	}

	var err error
	pkg.Archive, err = s.t.TranslatePackage(pkg.ImportPath, files, fileSet, s.importPackage)
	if err != nil {
		return err
	}

	if pkg.PkgObj == "" || pkg.IsCommand() {
		return nil
	}

	if err := s.writeLibraryPackage(pkg, pkg.PkgObj); err != nil {
		if strings.HasPrefix(pkg.PkgObj, build.Default.GOROOT) {
			// fall back to GOPATH
			if err := s.writeLibraryPackage(pkg, build.Default.GOPATH+pkg.PkgObj[len(build.Default.GOROOT):]); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	return nil
}

func (s *session) applyPatches(file *ast.File, fileSet *token.FileSet, importPath, basename string) *ast.File {
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

	case importPath == "runtime" && strings.HasPrefix(basename, "zgoarch_"):
		file, _ = parser.ParseFile(fileSet, basename, "package runtime\nconst theGoarch = `js`\n", 0)

	case importPath == "sync/atomic_test" && basename == "atomic_test.go":
		removeFunction("TestUnaligned64")

	case importPath == "text/template/parse" && basename == "lex.go": // this patch will be removed as soon as goroutines are supported
		file, _ = parser.ParseFile(fileSet, basename, "// Copyright 2011 The Go Authors. All rights reserved.\n// Use of this source code is governed by a BSD-style\n// license that can be found in the LICENSE file.\npackage parse\nimport (\n	\"container/list\"\n	\"fmt\"\n	\"strings\"\n	\"unicode\"\n	\"unicode/utf8\"\n)\n// item represents a token or text string returned from the scanner.\ntype item struct {\n	typ itemType // The type of this item.\n	pos Pos      // The starting position, in bytes, of this item in the input string.\n	val string   // The value of this item.\n}\nfunc (i item) String() string {\n	switch {\n	case i.typ == itemEOF:\n		return \"EOF\"\n	case i.typ == itemError:\n		return i.val\n	case i.typ > itemKeyword:\n		return fmt.Sprintf(\"<%s>\", i.val)\n	case len(i.val) > 10:\n		return fmt.Sprintf(\"%.10q...\", i.val)\n	}\n	return fmt.Sprintf(\"%q\", i.val)\n}\n// itemType identifies the type of lex items.\ntype itemType int\nconst (\n	itemError        itemType = iota // error occurred; value is text of error\n	itemBool                         // boolean constant\n	itemChar                         // printable ASCII character; grab bag for comma etc.\n	itemCharConstant                 // character constant\n	itemComplex                      // complex constant (1+2i); imaginary is just a number\n	itemColonEquals                  // colon-equals (':=') introducing a declaration\n	itemEOF\n	itemField      // alphanumeric identifier starting with '.'\n	itemIdentifier // alphanumeric identifier not starting with '.'\n	itemLeftDelim  // left action delimiter\n	itemLeftParen  // '(' inside action\n	itemNumber     // simple number, including imaginary\n	itemPipe       // pipe symbol\n	itemRawString  // raw quoted string (includes quotes)\n	itemRightDelim // right action delimiter\n	itemRightParen // ')' inside action\n	itemSpace      // run of spaces separating arguments\n	itemString     // quoted string (includes quotes)\n	itemText       // plain text\n	itemVariable   // variable starting with '$', such as '$' or  '$1' or '$hello'\n	// Keywords appear after all the rest.\n	itemKeyword  // used only to delimit the keywords\n	itemDot      // the cursor, spelled '.'\n	itemDefine   // define keyword\n	itemElse     // else keyword\n	itemEnd      // end keyword\n	itemIf       // if keyword\n	itemNil      // the untyped nil constant, easiest to treat as a keyword\n	itemRange    // range keyword\n	itemTemplate // template keyword\n	itemWith     // with keyword\n)\nvar key = map[string]itemType{\n	\".\":        itemDot,\n	\"define\":   itemDefine,\n	\"else\":     itemElse,\n	\"end\":      itemEnd,\n	\"if\":       itemIf,\n	\"range\":    itemRange,\n	\"nil\":      itemNil,\n	\"template\": itemTemplate,\n	\"with\":     itemWith,\n}\nconst eof = -1\n// stateFn represents the state of the scanner as a function that returns the next state.\ntype stateFn func(*lexer) stateFn\n// lexer holds the state of the scanner.\ntype lexer struct {\n	name       string     // the name of the input; used only for error reports\n	input      string     // the string being scanned\n	leftDelim  string     // start of action\n	rightDelim string     // end of action\n	state      stateFn    // the next lexing function to enter\n	pos        Pos        // current position in the input\n	start      Pos        // start position of this item\n	width      Pos        // width of last rune read from input\n	lastPos    Pos        // position of most recent item returned by nextItem\n	items      *list.List // scanned items\n	parenDepth int        // nesting depth of ( ) exprs\n}\n// next returns the next rune in the input.\nfunc (l *lexer) next() rune {\n	if int(l.pos) >= len(l.input) {\n		l.width = 0\n		return eof\n	}\n	r, w := utf8.DecodeRuneInString(l.input[l.pos:])\n	l.width = Pos(w)\n	l.pos += l.width\n	return r\n}\n// peek returns but does not consume the next rune in the input.\nfunc (l *lexer) peek() rune {\n	r := l.next()\n	l.backup()\n	return r\n}\n// backup steps back one rune. Can only be called once per call of next.\nfunc (l *lexer) backup() {\n	l.pos -= l.width\n}\n// emit passes an item back to the client.\nfunc (l *lexer) emit(t itemType) {\n	l.items.PushBack(item{t, l.start, l.input[l.start:l.pos]})\n	l.start = l.pos\n}\n// ignore skips over the pending input before this point.\nfunc (l *lexer) ignore() {\n	l.start = l.pos\n}\n// accept consumes the next rune if it's from the valid set.\nfunc (l *lexer) accept(valid string) bool {\n	if strings.IndexRune(valid, l.next()) >= 0 {\n		return true\n	}\n	l.backup()\n	return false\n}\n// acceptRun consumes a run of runes from the valid set.\nfunc (l *lexer) acceptRun(valid string) {\n	for strings.IndexRune(valid, l.next()) >= 0 {\n	}\n	l.backup()\n}\n// lineNumber reports which line we're on, based on the position of\n// the previous item returned by nextItem. Doing it this way\n// means we don't have to worry about peek double counting.\nfunc (l *lexer) lineNumber() int {\n	return 1 + strings.Count(l.input[:l.lastPos], \"\\n\")\n}\n// errorf returns an error token and terminates the scan by passing\n// back a nil pointer that will be the next state, terminating l.nextItem.\nfunc (l *lexer) errorf(format string, args ...interface{}) stateFn {\n	l.items.PushBack(item{itemError, l.start, fmt.Sprintf(format, args...)})\n	return nil\n}\n// nextItem returns the next item from the input.\nfunc (l *lexer) nextItem() item {\n	element := l.items.Front()\n	for element == nil {\n		l.state = l.state(l)\n		element = l.items.Front()\n	}\n	l.items.Remove(element)\n	item := element.Value.(item)\n	l.lastPos = item.pos\n	return item\n}\n// lex creates a new scanner for the input string.\nfunc lex(name, input, left, right string) *lexer {\n	if left == \"\" {\n		left = leftDelim\n	}\n	if right == \"\" {\n		right = rightDelim\n	}\n	l := &lexer{\n		name:       name,\n		input:      input,\n		leftDelim:  left,\n		rightDelim: right,\n		items:      list.New(),\n	}\n	l.state = lexText\n	return l\n}\n// state functions\nconst (\n	leftDelim    = \"{{\"\n	rightDelim   = \"}}\"\n	leftComment  = \"/*\"\n	rightComment = \"*/\"\n)\n// lexText scans until an opening action delimiter, \"{{\".\nfunc lexText(l *lexer) stateFn {\n	for {\n		if strings.HasPrefix(l.input[l.pos:], l.leftDelim) {\n			if l.pos > l.start {\n				l.emit(itemText)\n			}\n			return lexLeftDelim\n		}\n		if l.next() == eof {\n			break\n		}\n	}\n	// Correctly reached EOF.\n	if l.pos > l.start {\n		l.emit(itemText)\n	}\n	l.emit(itemEOF)\n	return nil\n}\n// lexLeftDelim scans the left delimiter, which is known to be present.\nfunc lexLeftDelim(l *lexer) stateFn {\n	l.pos += Pos(len(l.leftDelim))\n	if strings.HasPrefix(l.input[l.pos:], leftComment) {\n		return lexComment\n	}\n	l.emit(itemLeftDelim)\n	l.parenDepth = 0\n	return lexInsideAction\n}\n// lexComment scans a comment. The left comment marker is known to be present.\nfunc lexComment(l *lexer) stateFn {\n	l.pos += Pos(len(leftComment))\n	i := strings.Index(l.input[l.pos:], rightComment)\n	if i < 0 {\n		return l.errorf(\"unclosed comment\")\n	}\n	l.pos += Pos(i + len(rightComment))\n	if !strings.HasPrefix(l.input[l.pos:], l.rightDelim) {\n		return l.errorf(\"comment ends before closing delimiter\")\n	}\n	l.pos += Pos(len(l.rightDelim))\n	l.ignore()\n	return lexText\n}\n// lexRightDelim scans the right delimiter, which is known to be present.\nfunc lexRightDelim(l *lexer) stateFn {\n	l.pos += Pos(len(l.rightDelim))\n	l.emit(itemRightDelim)\n	return lexText\n}\n// lexInsideAction scans the elements inside action delimiters.\nfunc lexInsideAction(l *lexer) stateFn {\n	// Either number, quoted string, or identifier.\n	// Spaces separate arguments; runs of spaces turn into itemSpace.\n	// Pipe symbols separate and are emitted.\n	if strings.HasPrefix(l.input[l.pos:], l.rightDelim) {\n		if l.parenDepth == 0 {\n			return lexRightDelim\n		}\n		return l.errorf(\"unclosed left paren\")\n	}\n	switch r := l.next(); {\n	case r == eof || isEndOfLine(r):\n		return l.errorf(\"unclosed action\")\n	case isSpace(r):\n		return lexSpace\n	case r == ':':\n		if l.next() != '=' {\n			return l.errorf(\"expected :=\")\n		}\n		l.emit(itemColonEquals)\n	case r == '|':\n		l.emit(itemPipe)\n	case r == '\"':\n		return lexQuote\n	case r == '`':\n		return lexRawQuote\n	case r == '$':\n		return lexVariable\n	case r == '\\'':\n		return lexChar\n	case r == '.':\n		// special look-ahead for \".field\" so we don't break l.backup().\n		if l.pos < Pos(len(l.input)) {\n			r := l.input[l.pos]\n			if r < '0' || '9' < r {\n				return lexField\n			}\n		}\n		fallthrough // '.' can start a number.\n	case r == '+' || r == '-' || ('0' <= r && r <= '9'):\n		l.backup()\n		return lexNumber\n	case isAlphaNumeric(r):\n		l.backup()\n		return lexIdentifier\n	case r == '(':\n		l.emit(itemLeftParen)\n		l.parenDepth++\n		return lexInsideAction\n	case r == ')':\n		l.emit(itemRightParen)\n		l.parenDepth--\n		if l.parenDepth < 0 {\n			return l.errorf(\"unexpected right paren %#U\", r)\n		}\n		return lexInsideAction\n	case r <= unicode.MaxASCII && unicode.IsPrint(r):\n		l.emit(itemChar)\n		return lexInsideAction\n	default:\n		return l.errorf(\"unrecognized character in action: %#U\", r)\n	}\n	return lexInsideAction\n}\n// lexSpace scans a run of space characters.\n// One space has already been seen.\nfunc lexSpace(l *lexer) stateFn {\n	for isSpace(l.peek()) {\n		l.next()\n	}\n	l.emit(itemSpace)\n	return lexInsideAction\n}\n// lexIdentifier scans an alphanumeric.\nfunc lexIdentifier(l *lexer) stateFn {\nLoop:\n	for {\n		switch r := l.next(); {\n		case isAlphaNumeric(r):\n			// absorb.\n		default:\n			l.backup()\n			word := l.input[l.start:l.pos]\n			if !l.atTerminator() {\n				return l.errorf(\"bad character %#U\", r)\n			}\n			switch {\n			case key[word] > itemKeyword:\n				l.emit(key[word])\n			case word[0] == '.':\n				l.emit(itemField)\n			case word == \"true\", word == \"false\":\n				l.emit(itemBool)\n			default:\n				l.emit(itemIdentifier)\n			}\n			break Loop\n		}\n	}\n	return lexInsideAction\n}\n// lexField scans a field: .Alphanumeric.\n// The . has been scanned.\nfunc lexField(l *lexer) stateFn {\n	return lexFieldOrVariable(l, itemField)\n}\n// lexVariable scans a Variable: $Alphanumeric.\n// The $ has been scanned.\nfunc lexVariable(l *lexer) stateFn {\n	if l.atTerminator() { // Nothing interesting follows -> \"$\".\n		l.emit(itemVariable)\n		return lexInsideAction\n	}\n	return lexFieldOrVariable(l, itemVariable)\n}\n// lexVariable scans a field or variable: [.$]Alphanumeric.\n// The . or $ has been scanned.\nfunc lexFieldOrVariable(l *lexer, typ itemType) stateFn {\n	if l.atTerminator() { // Nothing interesting follows -> \".\" or \"$\".\n		if typ == itemVariable {\n			l.emit(itemVariable)\n		} else {\n			l.emit(itemDot)\n		}\n		return lexInsideAction\n	}\n	var r rune\n	for {\n		r = l.next()\n		if !isAlphaNumeric(r) {\n			l.backup()\n			break\n		}\n	}\n	if !l.atTerminator() {\n		return l.errorf(\"bad character %#U\", r)\n	}\n	l.emit(typ)\n	return lexInsideAction\n}\n// atTerminator reports whether the input is at valid termination character to\n// appear after an identifier. Breaks .X.Y into two pieces. Also catches cases\n// like \"$x+2\" not being acceptable without a space, in case we decide one\n// day to implement arithmetic.\nfunc (l *lexer) atTerminator() bool {\n	r := l.peek()\n	if isSpace(r) || isEndOfLine(r) {\n		return true\n	}\n	switch r {\n	case eof, '.', ',', '|', ':', ')', '(':\n		return true\n	}\n	// Does r start the delimiter? This can be ambiguous (with delim==\"//\", $x/2 will\n	// succeed but should fail) but only in extremely rare cases caused by willfully\n	// bad choice of delimiter.\n	if rd, _ := utf8.DecodeRuneInString(l.rightDelim); rd == r {\n		return true\n	}\n	return false\n}\n// lexChar scans a character constant. The initial quote is already\n// scanned. Syntax checking is done by the parser.\nfunc lexChar(l *lexer) stateFn {\nLoop:\n	for {\n		switch l.next() {\n		case '\\\\':\n			if r := l.next(); r != eof && r != '\\n' {\n				break\n			}\n			fallthrough\n		case eof, '\\n':\n			return l.errorf(\"unterminated character constant\")\n		case '\\'':\n			break Loop\n		}\n	}\n	l.emit(itemCharConstant)\n	return lexInsideAction\n}\n// lexNumber scans a number: decimal, octal, hex, float, or imaginary. This\n// isn't a perfect number scanner - for instance it accepts \".\" and \"0x0.2\"\n// and \"089\" - but when it's wrong the input is invalid and the parser (via\n// strconv) will notice.\nfunc lexNumber(l *lexer) stateFn {\n	if !l.scanNumber() {\n		return l.errorf(\"bad number syntax: %q\", l.input[l.start:l.pos])\n	}\n	if sign := l.peek(); sign == '+' || sign == '-' {\n		// Complex: 1+2i. No spaces, must end in 'i'.\n		if !l.scanNumber() || l.input[l.pos-1] != 'i' {\n			return l.errorf(\"bad number syntax: %q\", l.input[l.start:l.pos])\n		}\n		l.emit(itemComplex)\n	} else {\n		l.emit(itemNumber)\n	}\n	return lexInsideAction\n}\nfunc (l *lexer) scanNumber() bool {\n	// Optional leading sign.\n	l.accept(\"+-\")\n	// Is it hex?\n	digits := \"0123456789\"\n	if l.accept(\"0\") && l.accept(\"xX\") {\n		digits = \"0123456789abcdefABCDEF\"\n	}\n	l.acceptRun(digits)\n	if l.accept(\".\") {\n		l.acceptRun(digits)\n	}\n	if l.accept(\"eE\") {\n		l.accept(\"+-\")\n		l.acceptRun(\"0123456789\")\n	}\n	// Is it imaginary?\n	l.accept(\"i\")\n	// Next thing mustn't be alphanumeric.\n	if isAlphaNumeric(l.peek()) {\n		l.next()\n		return false\n	}\n	return true\n}\n// lexQuote scans a quoted string.\nfunc lexQuote(l *lexer) stateFn {\nLoop:\n	for {\n		switch l.next() {\n		case '\\\\':\n			if r := l.next(); r != eof && r != '\\n' {\n				break\n			}\n			fallthrough\n		case eof, '\\n':\n			return l.errorf(\"unterminated quoted string\")\n		case '\"':\n			break Loop\n		}\n	}\n	l.emit(itemString)\n	return lexInsideAction\n}\n// lexRawQuote scans a raw quoted string.\nfunc lexRawQuote(l *lexer) stateFn {\nLoop:\n	for {\n		switch l.next() {\n		case eof, '\\n':\n			return l.errorf(\"unterminated raw quoted string\")\n		case '`':\n			break Loop\n		}\n	}\n	l.emit(itemRawString)\n	return lexInsideAction\n}\n// isSpace reports whether r is a space character.\nfunc isSpace(r rune) bool {\n	return r == ' ' || r == '\\t'\n}\n// isEndOfLine reports whether r is an end-of-line character.\nfunc isEndOfLine(r rune) bool {\n	return r == '\\r' || r == '\\n'\n}\n// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.\nfunc isAlphaNumeric(r rune) bool {\n	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)\n}", 0)
	}

	return file
}

func (s *session) writeLibraryPackage(pkg *packageData, pkgObj string) error {
	if err := os.MkdirAll(filepath.Dir(pkgObj), 0777); err != nil {
		return err
	}

	data, err := s.t.WriteArchive(pkg.Archive)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(pkgObj, data, 0666)
}

func (s *session) writeCommandPackage(pkg *packageData, pkgObj string) error {
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
	mapFile, err := os.Create(pkgObj + ".map")
	if err != nil {
		return err
	}
	defer mapFile.Close()

	var allPkgs []*translator.Archive
	for _, depPath := range pkg.Archive.Dependencies {
		dep, err := s.importPackage(depPath)
		if err != nil {
			return err
		}
		allPkgs = append(allPkgs, dep)
	}

	m := sourcemap.Map{File: filepath.Base(pkgObj)}
	s.t.WriteProgramCode(allPkgs, pkg.ImportPath, &translator.SourceMapFilter{Writer: codeFile, MappingCallback: func(generatedLine, generatedColumn int, fileSet *token.FileSet, originalPos token.Pos) {
		pos := fileSet.Position(originalPos)
		file := pos.Filename
		switch {
		case strings.HasPrefix(file, build.Default.GOPATH):
			file = filepath.ToSlash(filepath.Join("gopath", file[len(build.Default.GOPATH):]))
		case strings.HasPrefix(file, build.Default.GOROOT):
			file = filepath.ToSlash(filepath.Join("goroot", file[len(build.Default.GOROOT):]))
		default:
			file = filepath.Base(file)
		}
		m.AddMapping(&sourcemap.Mapping{GeneratedLine: generatedLine, GeneratedColumn: generatedColumn, OriginalFile: file, OriginalLine: pos.Line, OriginalColumn: pos.Column})
	}})
	fmt.Fprintf(codeFile, "//# sourceMappingURL=%s.map\n", filepath.Base(pkgObj))
	m.WriteTo(mapFile)

	return nil
}

func runNode(script string, args []string, dir string) error {
	node := exec.Command("node", append([]string{script}, args...)...)
	node.Dir = dir
	node.Stdin = os.Stdin
	node.Stdout = os.Stdout
	node.Stderr = os.Stderr
	return node.Run()
}
