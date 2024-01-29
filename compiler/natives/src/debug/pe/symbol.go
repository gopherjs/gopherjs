//go:build js
// +build js

package pe

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"io"
)

func copyToAuxFormat5(sym *COFFSymbol) *COFFSymbolAuxFormat5 {
	check := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	order := binary.LittleEndian
	buf := &bytes.Buffer{}
	check(binary.Write(buf, order, sym.Name[:]))
	check(binary.Write(buf, order, sym.Value))
	check(binary.Write(buf, order, sym.SectionNumber))
	check(binary.Write(buf, order, sym.Type))
	check(binary.Write(buf, order, sym.StorageClass))
	check(binary.Write(buf, order, sym.NumberOfAuxSymbols))

	aux := &pe.COFFSymbolAuxFormat5{}
	check(binary.Read(buf, order, &aux.Size))
	check(binary.Read(buf, order, &aux.NumRelocs))
	check(binary.Read(buf, order, &aux.NumLineNumbers))
	check(binary.Read(buf, order, &aux.Checksum))
	check(binary.Read(buf, order, &aux.SecNum))
	check(binary.Read(buf, order, &aux.Selection))
	return aux
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
			// The following was reading one struct as another struct with
			// the same memory footprint. This doesn't work in JS so the
			// `rv` value is left with a bunch of `undefined`s. So replace
			// raux := (*COFFSymbolAuxFormat5)(unsafe.Pointer(&syms[k]))
			// (an in memory remap) with the following copy.
			aux := copyToAuxFormat5(&syms[k])
			err = binary.Read(r, binary.LittleEndian, aux)
			if err != nil {
				return nil, fmt.Errorf("fail to read symbol table: %v", err)
			}
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
	rv = copyToAuxFormat5(pesymn)
	return rv, nil
}
