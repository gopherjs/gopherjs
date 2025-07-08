package sourceWriter

import (
	"bytes"
	"encoding/binary"
	"go/token"
	"io"
)

type MappingCallbackHandle func(generatedLine, generatedColumn int, originalPos token.Position)

func EncodePos(pos token.Pos) []byte {
	result := make([]byte, 5)
	result[0] = posPrefix
	binary.BigEndian.PutUint32(result[1:], uint32(pos))
	return result
}

func decodePos(p []byte) token.Pos {
	if len(p) < 4 {
		return token.NoPos
	}
	return token.Pos(binary.BigEndian.Uint32(p))
}

const posPrefix = '\b'

var _ io.Writer = (*SourceWriter)(nil)

type sourceMapFilter struct {
	out             io.Writer
	mappingCallback MappingCallbackHandle
	line            int
	column          int
	fileSet         *token.FileSet
}

func newSourceMapFilter(w io.Writer, mappingCallback MappingCallbackHandle) *sourceMapFilter {
	return &sourceMapFilter{
		out:             w,
		mappingCallback: mappingCallback,
	}
}

func (f *sourceMapFilter) SetFileSet(fileSet *token.FileSet) {
	if f.mappingCallback != nil && fileSet != nil {
		f.fileSet = fileSet
	}
}

func (f *sourceMapFilter) Write(p []byte) (n int, err error) {
	var n2 int
	for {
		i := bytes.IndexByte(p, posPrefix)
		w := p
		if i != -1 {
			w = p[:i]
		}

		n2, err = f.out.Write(w)
		n += n2
		if err != nil || i < 0 {
			return n, err
		}

		f.updateOutputPos(w)
		f.writePosToMapping(p[i:])
		p = p[i+5:]
		n += 5
	}
}

func (f *sourceMapFilter) updateOutputPos(w []byte) {
	for {
		i := bytes.IndexByte(w, '\n')
		if i < 0 {
			f.column += len(w)
			break
		}
		f.line++
		f.column = 0
		w = w[i+1:]
	}
}

func (f *sourceMapFilter) writePosToMapping(p []byte) {
	if f.mappingCallback != nil {
		pos := decodePos(p)
		f.mappingCallback(f.line+1, f.column, f.fileSet.Position(pos))
	}
}
