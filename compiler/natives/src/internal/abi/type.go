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
