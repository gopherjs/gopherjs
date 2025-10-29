package sourcemapx

import (
	"bytes"
	"fmt"
	"go/token"
	"io"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/neelance/sourcemap"
	log "github.com/sirupsen/logrus"
)

type (
	goMappingCallbackHandle func(generatedLine, generatedColumn int, originalPos token.Position, originalName string)
	jsMappingCallbackHandle func(isolated *sourcemap.Mapping)
)

// Filter implements io.Writer which extracts source map hints from the written
// stream and passed them to the MappingCallback if it's not nil. Encoded hints
// are always filtered out of the output stream.
type Filter struct {
	Writer            io.Writer
	FileSet           *token.FileSet
	goMappingCallback goMappingCallbackHandle
	jsMappingCallback jsMappingCallbackHandle

	m        *sourcemap.Map
	goroot   string
	gopath   string
	localMap bool

	line   int
	column int
}

func (f *Filter) EnableMapping(jsFileName, goroot, gopath string, localMap bool) {
	f.m = &sourcemap.Map{File: jsFileName}
	f.goroot = goroot
	f.gopath = gopath
	f.localMap = localMap
	f.goMappingCallback = f.defaultGoMappingCallback
	f.jsMappingCallback = f.defaultJSMappingCallback
}

func (f *Filter) IsMapping() bool {
	return f.goMappingCallback != nil || f.jsMappingCallback != nil
}

func (f *Filter) WriteMappingTo(w io.Writer) error {
	return f.m.WriteTo(w)
}

func (f *Filter) Write(p []byte) (n int, err error) {
	var n2 int
	for {
		i := FindHint(p)
		w := p
		if i != -1 {
			w = p[:i]
		}

		n2, err = f.Writer.Write(w)
		n += n2
		for {
			i := bytes.IndexByte(w, '\n')
			if i == -1 {
				f.column += len(w)
				break
			}
			f.line++
			f.column = 0
			w = w[i+1:]
		}

		if err != nil || i == -1 {
			return
		}
		h, length := ReadHint(p[i:])
		if f.goMappingCallback != nil {
			value, err := h.Unpack()
			if err != nil {
				panic(fmt.Errorf("failed to unpack source map hint: %w", err))
			}
			switch value := value.(type) {
			case token.Pos:
				f.goMappingCallback(f.line+1, f.column, f.FileSet.Position(value), "")
			case Identifier:
				f.goMappingCallback(f.line+1, f.column, f.FileSet.Position(value.OriginalPos), value.OriginalName)
			default:
				panic(fmt.Errorf("unexpected source map hint type: %T", value))
			}
		}
		p = p[i+length:]
		n += length
	}
}

func (f *Filter) WriteJS(jsSource, jsFilePath string, minify bool) (n int, err error) {
	if !minify && f.jsMappingCallback == nil {
		// If not minimifying and not mapping, write source as-is.
		return f.Write([]byte(jsSource))
	}

	options := api.TransformOptions{
		Target:         api.ES2015,
		Charset:        api.CharsetUTF8,
		LegalComments:  api.LegalCommentsEndOfFile,
		JSX:            api.JSXPreserve,
		JSXSideEffects: true,
		TreeShaking:    api.TreeShakingFalse,
	}

	if minify {
		options.MinifyWhitespace = true
		options.MinifyIdentifiers = true
		options.MinifySyntax = true
		options.KeepNames = true
	}

	if f.jsMappingCallback != nil {
		options.Sourcefile = jsFilePath
		options.Sourcemap = api.SourceMapExternal
		options.SourcesContent = api.SourcesContentExclude
	}

	result := api.Transform(jsSource, options)
	for _, w := range result.Warnings {
		log.Warnf("%d:%d: %s\n%s\n", w.Location.Line, w.Location.Column, w.Text, w.Location.LineText)
	}
	if errCount := len(result.Errors); errCount > 0 {
		for _, e := range result.Errors {
			log.Errorf("%d:%d: %s\n%s\n", e.Location.Line, e.Location.Column, e.Text, e.Location.LineText)
		}
		log.Fatalf(`JS minification failed with %d errors`, errCount)
	}

	if f.jsMappingCallback != nil {
		sm, err := sourcemap.ReadFrom(bytes.NewReader(result.Map))
		if err != nil {
			return 0, fmt.Errorf(`failed to read source map: %w`, err)
		}
		mappings := sm.DecodedMappings()
		for _, mapping := range mappings {
			f.jsMappingCallback(mapping)
		}
	}

	return f.Write(result.Code)
}

// defaultGoMappingCallback is the default callback for source map generatio for Go sources.
func (f *Filter) defaultGoMappingCallback(generatedLine, generatedColumn int, originalPos token.Position, originalName string) {
	mapping := &sourcemap.Mapping{GeneratedLine: generatedLine, GeneratedColumn: generatedColumn}

	if originalPos.IsValid() {
		mapping.OriginalFile = f.normalizePath(originalPos.Filename)
		mapping.OriginalLine = originalPos.Line
		mapping.OriginalColumn = originalPos.Column
	}

	if originalName != "" {
		mapping.OriginalName = originalName
	}

	f.m.AddMapping(mapping)
}

// defaultJSMappingCallback is the default callback for source map generatio for JS sources.
// The given mapping is from the JS file isolated and needs to be adjusted
// before adding to the filter's source map.
func (f *Filter) defaultJSMappingCallback(isolated *sourcemap.Mapping) {
	isolated.OriginalFile = f.normalizePath(isolated.OriginalFile)

	// Adjust line and column numbers to account for existing offset.
	if isolated.GeneratedLine == 0 {
		isolated.GeneratedColumn += f.column
	}
	isolated.GeneratedLine += f.line

	// Remove original names since esbuild causes an issue in some cases where
	// a parameter name is used instead of the function name.
	// For example, `var $panic = value => { ... };` from goroutine.js will
	// be minimized to `$panic=a(e=>{ ... },"$panic")` where `a` is
	// Object.defineProperty added by esbuild and `e` is the minimized `value`.
	// They write `value` to the source map but that causes JS to use `value`
	// instead of `$panic` when viewing the stack trace.
	// This may be a configuration issue with esbuild, but for now we just
	// remove the "original names" and allow JS to fall back to the preserved
	// function names to avoid this problem.
	isolated.OriginalName = ``

	// Add mapping to the filter's source map.
	f.m.AddMapping(isolated)
}

func (f *Filter) normalizePath(file string) string {
	switch hasGopathPrefix, prefixLen := hasGopathPrefix(file, f.gopath); {
	case f.localMap:
		// no-op:  keep file as-is
		return file
	case hasGopathPrefix:
		return filepath.ToSlash(file[prefixLen+4:])
	case strings.HasPrefix(file, f.goroot):
		return filepath.ToSlash(file[len(f.goroot)+4:])
	default:
		return filepath.Base(file)
	}
}

// hasGopathPrefix returns true and the length of the matched GOPATH workspace,
// iff file has a prefix that matches one of the GOPATH workspaces.
func hasGopathPrefix(file, gopath string) (hasGopathPrefix bool, prefixLen int) {
	gopathWorkspaces := filepath.SplitList(gopath)
	for _, gopathWorkspace := range gopathWorkspaces {
		gopathWorkspace = filepath.Clean(gopathWorkspace)
		if strings.HasPrefix(file, gopathWorkspace) {
			return true, len(gopathWorkspace)
		}
	}
	return false, 0
}
