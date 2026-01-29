package abi

var UncommonTypeMap = make(map[*Type]*UncommonType)

func (t *Type) Uncommon() *UncommonType {
	return UncommonTypeMap[t]
}

type UncommonType struct {
	PkgPath  NameOff // import path
	Mcount   uint16  // method count
	Xcount   uint16  // exported method count
	Methods_ []Method
}

func (t *UncommonType) Methods() []Method {
	return t.Methods_
}

func (t *UncommonType) ExportedMethods() []Method {
	return t.Methods_[:t.Xcount:t.Xcount]
}

type FuncType struct {
	Type     `reflect:"func"`
	InCount  uint16
	OutCount uint16

	In_  []*Type
	Out_ []*Type
}

func (t *FuncType) InSlice() []*Type {
	return t.In_
}

func (t *FuncType) OutSlice() []*Type {
	return t.Out_
}
