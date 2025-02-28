package jsFile

import (
	"fmt"
	"go/build"
	"io"
	"strings"
	"time"

	"golang.org/x/tools/go/buildutil"
)

// JSFile represents a *.inc.js file metadata and content.
type JSFile struct {
	Path    string // Full file path for the build context the file came from.
	ModTime time.Time
	Content []byte
}

// JSFilesFromDir finds and loads any *.inc.js packages in the build context
// directory.
func JSFilesFromDir(bctx *build.Context, dir string) ([]JSFile, error) {
	files, err := buildutil.ReadDir(bctx, dir)
	if err != nil {
		return nil, err
	}
	var jsFiles []JSFile
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".inc.js") || file.IsDir() {
			continue
		}
		if file.Name()[0] == '_' || file.Name()[0] == '.' {
			continue // Skip "hidden" files that are typically ignored by the Go build system.
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

		jsFiles = append(jsFiles, JSFile{
			Path:    path,
			ModTime: file.ModTime(),
			Content: content,
		})
	}
	return jsFiles, nil
}
