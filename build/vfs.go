package build

import (
	"io"
	"net/http"
	"os"
	"strings"
)

// vfs is a convenience wrapper around http.FileSystem that provides accessor
// methods required by go/build.Context.
type vfs struct{ http.FileSystem }

func (fs vfs) IsDir(name string) bool {
	dir, err := fs.Open(name)
	if err != nil {
		return false
	}
	defer dir.Close()
	info, err := dir.Stat()
	if err != nil {
		return false
	}
	return info.IsDir()
}

func (vfs) HasSubDir(root, name string) (rel string, ok bool) {
	panic("vfs.HasSubDir() is not implemented")
}

func (fs vfs) ReadDir(name string) (fi []os.FileInfo, err error) {
	dir, err := fs.Open(name)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	return dir.Readdir(0)
}

func (fs vfs) OpenFile(name string) (r io.ReadCloser, err error) {
	return fs.Open(name)
}

func splitPathList(list string) []string {
	if list == "" {
		return nil
	}
	const pathListSeparator = ":" // UNIX style
	return strings.Split(list, pathListSeparator)
}

// withPrefix implements http.FileSystem, which places the underlying FS under
// the given prefix path.
type withPrefix struct {
	fs     http.FileSystem
	prefix string
}

func (wp *withPrefix) Open(name string) (http.File, error) {
	if !strings.HasPrefix(name, wp.prefix) {
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
	}
	f, err := wp.fs.Open(strings.TrimPrefix(name, wp.prefix))
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: name, Err: err}
	}
	return f, nil
}
