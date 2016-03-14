package sourcemap

import (
	"bytes"
	"encoding/json"
	"io"
	"sort"
	"strings"
)

type Map struct {
	Version         int      `json:"version"`
	File            string   `json:"file,omitempty"`
	SourceRoot      string   `json:"sourceRoot,omitempty"`
	Sources         []string `json:"sources,omitempty"`
	Names           []string `json:"names,omitempty"`
	Mappings        string   `json:"mappings"`
	decodedMappings []*Mapping
}

type Mapping struct {
	GeneratedLine   int
	GeneratedColumn int
	OriginalFile    string
	OriginalLine    int
	OriginalColumn  int
	OriginalName    string
}

func ReadFrom(r io.Reader) (*Map, error) {
	d := json.NewDecoder(r)
	var m Map
	if err := d.Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

const base64encode = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

var base64decode [256]int

func init() {
	for i := 0; i < len(base64decode); i++ {
		base64decode[i] = 0xff
	}
	for i := 0; i < len(base64encode); i++ {
		base64decode[base64encode[i]] = i
	}
}

func (m *Map) decodeMappings() {
	if m.decodedMappings != nil {
		return
	}

	r := strings.NewReader(m.Mappings)
	var generatedLine = 1
	var generatedColumn = 0
	var originalFile = 0
	var originalLine = 1
	var originalColumn = 0
	var originalName = 0
	for r.Len() != 0 {
		b, _ := r.ReadByte()
		if b == ',' {
			continue
		}
		if b == ';' {
			generatedLine++
			generatedColumn = 0
			continue
		}
		r.UnreadByte()

		count := 0
		readVLQ := func() int {
			v := 0
			s := uint(0)
			for {
				b, _ := r.ReadByte()
				o := base64decode[b]
				if o == 0xff {
					r.UnreadByte()
					return 0
				}
				v += (o &^ 32) << s
				if o&32 == 0 {
					break
				}
				s += 5
			}
			count++
			if v&1 != 0 {
				return -(v >> 1)
			}
			return v >> 1
		}
		generatedColumn += readVLQ()
		originalFile += readVLQ()
		originalLine += readVLQ()
		originalColumn += readVLQ()
		originalName += readVLQ()

		switch count {
		case 1:
			m.decodedMappings = append(m.decodedMappings, &Mapping{generatedLine, generatedColumn, "", 0, 0, ""})
		case 4:
			m.decodedMappings = append(m.decodedMappings, &Mapping{generatedLine, generatedColumn, m.Sources[originalFile], originalLine, originalColumn, ""})
		case 5:
			m.decodedMappings = append(m.decodedMappings, &Mapping{generatedLine, generatedColumn, m.Sources[originalFile], originalLine, originalColumn, m.Names[originalName]})
		}
	}
}

func (m *Map) DecodedMappings() []*Mapping {
	m.decodeMappings()
	return m.decodedMappings
}

func (m *Map) ClearMappings() {
	m.Mappings = ""
	m.decodedMappings = nil
}

func (m *Map) AddMapping(mapping *Mapping) {
	m.decodedMappings = append(m.decodedMappings, mapping)
}

func (m *Map) Len() int {
	m.decodeMappings()
	return len(m.DecodedMappings())
}

func (m *Map) Less(i, j int) bool {
	a := m.decodedMappings[i]
	b := m.decodedMappings[j]
	return a.GeneratedLine < b.GeneratedLine || (a.GeneratedLine == b.GeneratedLine && a.GeneratedColumn < b.GeneratedColumn)
}

func (m *Map) Swap(i, j int) {
	m.decodedMappings[i], m.decodedMappings[j] = m.decodedMappings[j], m.decodedMappings[i]
}

func (m *Map) EncodeMappings() {
	sort.Sort(m)
	m.Sources = nil
	fileIndexMap := make(map[string]int)
	m.Names = nil
	nameIndexMap := make(map[string]int)
	var generatedLine = 1
	var generatedColumn = 0
	var originalFile = 0
	var originalLine = 1
	var originalColumn = 0
	var originalName = 0
	buf := bytes.NewBuffer(nil)
	comma := false
	for _, mapping := range m.decodedMappings {
		for mapping.GeneratedLine > generatedLine {
			buf.WriteByte(';')
			generatedLine++
			generatedColumn = 0
			comma = false
		}
		if comma {
			buf.WriteByte(',')
		}

		writeVLQ := func(v int) {
			v <<= 1
			if v < 0 {
				v = -v
				v |= 1
			}
			for v >= 32 {
				buf.WriteByte(base64encode[32|(v&31)])
				v >>= 5
			}
			buf.WriteByte(base64encode[v])
		}

		writeVLQ(mapping.GeneratedColumn - generatedColumn)
		generatedColumn = mapping.GeneratedColumn

		if mapping.OriginalFile != "" {
			fileIndex, ok := fileIndexMap[mapping.OriginalFile]
			if !ok {
				fileIndex = len(m.Sources)
				fileIndexMap[mapping.OriginalFile] = fileIndex
				m.Sources = append(m.Sources, mapping.OriginalFile)
			}
			writeVLQ(fileIndex - originalFile)
			originalFile = fileIndex

			writeVLQ(mapping.OriginalLine - originalLine)
			originalLine = mapping.OriginalLine

			writeVLQ(mapping.OriginalColumn - originalColumn)
			originalColumn = mapping.OriginalColumn

			if mapping.OriginalName != "" {
				nameIndex, ok := nameIndexMap[mapping.OriginalName]
				if !ok {
					nameIndex = len(m.Names)
					nameIndexMap[mapping.OriginalName] = nameIndex
					m.Names = append(m.Names, mapping.OriginalName)
				}
				writeVLQ(nameIndex - originalName)
				originalName = nameIndex
			}
		}

		comma = true
	}
	m.Mappings = buf.String()
}

func (m *Map) WriteTo(w io.Writer) error {
	if m.Version == 0 {
		m.Version = 3
	}
	if m.decodedMappings != nil {
		m.EncodeMappings()
	}
	enc := json.NewEncoder(w)
	return enc.Encode(m)
}
