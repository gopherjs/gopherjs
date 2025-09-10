//go:build js

package pe

import (
	"encoding/binary"
	"fmt"
	"io"
)

// bytesBufferLite is a simplified bytes.Buffer to avoid
// including `bytes` as a new import into the pe package.
type bytesBufferLite struct {
	data []byte
	off  int
}

func (buf *bytesBufferLite) Write(p []byte) (int, error) {
	buf.data = append(buf.data, p...)
	return len(p), nil
}

func (buf *bytesBufferLite) Read(p []byte) (int, error) {
	n := copy(p, buf.data[buf.off:])
	buf.off += n
	return n, nil
}

func copyToAuxFormat5(sym *COFFSymbol) (*COFFSymbolAuxFormat5, error) {
	buf := &bytesBufferLite{data: make([]byte, 0, 20)}
	if err := binary.Write(buf, binary.LittleEndian, sym); err != nil {
		return nil, err
	}
	aux := &COFFSymbolAuxFormat5{}
	if err := binary.Read(buf, binary.LittleEndian, aux); err != nil {
		return nil, err
	}
	return aux, nil
}

func copyFromAuxFormat5(aux *COFFSymbolAuxFormat5) (*COFFSymbol, error) {
	buf := &bytesBufferLite{data: make([]byte, 0, 20)}
	if err := binary.Write(buf, binary.LittleEndian, aux); err != nil {
		return nil, err
	}
	sym := &COFFSymbol{}
	if err := binary.Read(buf, binary.LittleEndian, sym); err != nil {
		return nil, err
	}
	return sym, nil
}

func readCOFFSymbols(fh *FileHeader, r io.ReadSeeker) ([]COFFSymbol, error) {
	if fh.PointerToSymbolTable == 0 {
		return nil, nil
	}
	if fh.NumberOfSymbols <= 0 {
		return nil, nil
	}
	_, err := r.Seek(int64(fh.PointerToSymbolTable), seekStart)
	if err != nil {
		return nil, fmt.Errorf("fail to seek to symbol table: %v", err)
	}
	syms := make([]COFFSymbol, fh.NumberOfSymbols)
	naux := 0
	for k := range syms {
		if naux == 0 {
			err = binary.Read(r, binary.LittleEndian, &syms[k])
			if err != nil {
				return nil, fmt.Errorf("fail to read symbol table: %v", err)
			}
			naux = int(syms[k].NumberOfAuxSymbols)
		} else {
			naux--
			// The following was reading into one struct with the same memory
			// footprint as another struck. This doesn't work in JS so the
			// `syms` value is left with a bunch of defaults. So replace
			// aux := (*COFFSymbolAuxFormat5)(unsafe.Pointer(&syms[k]))
			// (an in memory remap) with the following read and then copy.
			aux := &COFFSymbolAuxFormat5{}
			err = binary.Read(r, binary.LittleEndian, aux)
			if err != nil {
				return nil, fmt.Errorf("fail to read symbol table: %v", err)
			}
			pesymn, err := copyFromAuxFormat5(aux)
			if err != nil {
				return nil, err
			}
			syms[k] = *pesymn
		}
	}
	if naux != 0 {
		return nil, fmt.Errorf("fail to read symbol table: %d aux symbols unread", naux)
	}
	return syms, nil
}

func (f *File) COFFSymbolReadSectionDefAux(idx int) (*COFFSymbolAuxFormat5, error) {
	var rv *COFFSymbolAuxFormat5
	if idx < 0 || idx >= len(f.COFFSymbols) {
		return rv, fmt.Errorf("invalid symbol index")
	}
	pesym := &f.COFFSymbols[idx]
	const IMAGE_SYM_CLASS_STATIC = 3
	if pesym.StorageClass != uint8(IMAGE_SYM_CLASS_STATIC) {
		return rv, fmt.Errorf("incorrect symbol storage class")
	}
	if pesym.NumberOfAuxSymbols == 0 || idx+1 >= len(f.COFFSymbols) {
		return rv, fmt.Errorf("aux symbol unavailable")
	}
	pesymn := &f.COFFSymbols[idx+1]
	// The following was reading one struct as another struct with
	// the same memory footprint. This doesn't work in JS so the
	// `rv` value is left with a bunch of `undefined`s. So replace
	// rv = (*COFFSymbolAuxFormat5)(unsafe.Pointer(pesymn))
	// (an in memory remap) with the following copy.
	return copyToAuxFormat5(pesymn)
}
