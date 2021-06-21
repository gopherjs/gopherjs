package build

import (
	"io"
	"net/http"
	"os"
	"path"
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

// hasSubdir reports whether dir is lexically a subdirectory of
// root, perhaps multiple levels below. It does not try to check
// whether dir exists.
// If so, hasSubdir sets rel to a slash-separated path that
// can be joined to root to produce a path equivalent to dir.
func hasSubdir(root, dir string) (rel string, ok bool) {
	// Implementation based on golang.org/x/tools/go/buildutil.
	const sep = "/" // UNIX style
	root = path.Clean(root)
	if !strings.HasSuffix(root, sep) {
		root += sep
	}

	dir = path.Clean(dir)
	if !strings.HasPrefix(dir, root) {
		return "", false
	}

	return dir[len(root):], true
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
