package incjs

import (
	"fmt"
	"go/build"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/tools/go/buildutil"
)

// Ext is the file extension for .inc.js files.
const Ext = ".inc.js"

// File represents a *.inc.js file metadata and content.
type File struct {
	Path    string // Full file path for the build context the file came from.
	ModTime time.Time
	Content []byte
}

// FromDir finds and loads any *.inc.js packages in the build context
// directory.
func FromDir(bctx *build.Context, dir string) ([]File, error) {
	files, err := buildutil.ReadDir(bctx, dir)
	if err != nil {
		return nil, err
	}
	var jsFiles []File
	for _, file := range files {
		f, err := fromFileInfo(bctx, dir, file)
		if err != nil {
			return nil, err
		}
		if f != nil {
			jsFiles = append(jsFiles, *f)
		}
	}
	return jsFiles, nil
}

func isIncJS(filename string) bool {
	return strings.HasSuffix(filename, Ext)
}

func fromFileInfo(bctx *build.Context, dir string, file os.FileInfo) (*File, error) {
	if !isIncJS(file.Name()) || file.IsDir() {
		return nil, nil
	}
	if file.Name()[0] == '_' || file.Name()[0] == '.' {
		return nil, nil // Skip "hidden" files that are typically ignored by the Go build system.
	}

	path := buildutil.JoinPath(bctx, dir, file.Name())
	f, err := buildutil.OpenFile(bctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s from %v: %w", path, bctx, err)
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s from %v: %w", path, bctx, err)
	}

	return &File{
		Path:    path,
		ModTime: file.ModTime(),
		Content: content,
	}, err
}

func FromFilename(filename string) (*File, error) {
	if !isIncJS(filename) {
		return nil, nil
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", filename, err)
	}

	info, err := os.Stat(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %s: %w", filename, err)
	}

	return &File{
		Path:    filename,
		ModTime: info.ModTime(),
		Content: content,
	}, nil
}
