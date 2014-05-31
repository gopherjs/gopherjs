package build

import (
	"bitbucket.org/kardianos/osext"
	"code.google.com/p/go.exp/fsnotify"
	"fmt"
	"github.com/gopherjs/gopherjs/compiler"
	"github.com/neelance/sourcemap"
	"go/build"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Options struct {
	GOROOT        string
	GOPATH        string
	Verbose       bool
	Watch         bool
	CreateMapFile bool
	Minify        bool
}

func (o *Options) Normalize() {
	if o.GOROOT == "" {
		o.GOROOT = build.Default.GOROOT
	}

	if o.GOPATH == "" {
		o.GOPATH = build.Default.GOPATH
	}

	o.Verbose = o.Verbose || o.Watch
}

type PackageData struct {
	*build.Package
	SrcModTime time.Time
	UpToDate   bool
	Archive    *compiler.Archive
}

type Session struct {
	T        *compiler.Compiler
	Packages map[string]*PackageData
	options  *Options
	Watcher  *fsnotify.Watcher
}

func NewSession(options *Options) *Session {
	s := &Session{
		T:        compiler.New(),
		options:  options,
		Packages: make(map[string]*PackageData),
	}
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
	buildContext := &build.Context{
		GOROOT:   s.options.GOROOT,
		GOPATH:   s.options.GOPATH,
		GOOS:     build.Default.GOOS,
		GOARCH:   s.ArchSuffix(),
		Compiler: "gc",
	}
	if s.Watcher != nil {
		s.Watcher.Watch(packagePath)
	}
	buildPkg, err := buildContext.ImportDir(packagePath, 0)
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

	buildPkg, err := compiler.Import(path, build.AllowBinary, s.ArchSuffix())
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
			if importedPkgPath == "unsafe" {
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

			pkg.Archive, err = s.T.UnmarshalArchive(pkg.PkgObj, pkg.ImportPath, objFile)
			if err != nil {
				return err
			}

			return nil
		}
	}

	fileSet := token.NewFileSet()
	files, err := compiler.Parse(pkg.Package, fileSet)
	if err != nil {
		return err
	}
	pkg.Archive, err = s.T.Compile(pkg.ImportPath, files, fileSet, s.ImportPackage)
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
			// fall back to GOPATH
			if err := s.writeLibraryPackage(pkg, s.options.GOPATH+pkg.PkgObj[len(s.options.GOROOT):]); err != nil {
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

	data, err := s.T.MarshalArchive(pkg.Archive)
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

	var allPkgs []*compiler.Archive
	for _, depPath := range pkg.Archive.Dependencies {
		dep, err := s.ImportPackage(string(depPath))
		if err != nil {
			return err
		}
		allPkgs = append(allPkgs, dep)
	}

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
			switch {
			case strings.HasPrefix(file, s.options.GOPATH):
				file = filepath.ToSlash(filepath.Join("/gopath", file[len(s.options.GOPATH):]))
			case strings.HasPrefix(file, s.options.GOROOT):
				file = filepath.ToSlash(filepath.Join("/goroot", file[len(s.options.GOROOT):]))
			default:
				file = filepath.Base(file)
			}
			m.AddMapping(&sourcemap.Mapping{GeneratedLine: generatedLine, GeneratedColumn: generatedColumn, OriginalFile: file, OriginalLine: pos.Line, OriginalColumn: pos.Column})
		}
	}

	s.T.WriteProgramCode(allPkgs, pkg.ImportPath, sourceMapFilter)

	return nil
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
