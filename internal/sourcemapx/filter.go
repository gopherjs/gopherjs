package sourcemapx

import (
	"bytes"
	"fmt"
	"go/token"
	"io"
)

// Filter implements io.Writer which extracts source map hints from the written
// stream and passed them to the MappingCallback if it's not nil. Encoded hints
// are always filtered out of the output stream.
type Filter struct {
	Writer          io.Writer
	FileSet         *token.FileSet
	MappingCallback func(generatedLine, generatedColumn int, originalPos token.Position, originalName string)

	line   int
	column int
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
		if f.MappingCallback != nil {
			value, err := h.Unpack()
			if err != nil {
				panic(fmt.Errorf("failed to unpack source map hint: %w", err))
			}
			switch value := value.(type) {
			case token.Pos:
				f.MappingCallback(f.line+1, f.column, f.FileSet.Position(value), "")
			case Identifier:
				f.MappingCallback(f.line+1, f.column, f.FileSet.Position(value.OriginalPos), value.OriginalName)
			default:
				panic(fmt.Errorf("unexpected source map hint type: %T", value))
			}
		}
		p = p[i+length:]
		n += length
	}
}
