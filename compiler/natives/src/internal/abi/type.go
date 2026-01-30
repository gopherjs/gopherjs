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

type Name struct {
	name     string
	tag      string
	exported bool
	embedded bool
}

func (n Name) IsExported() bool { return n.exported }
func (n Name) HasTag() bool     { return len(n.tag) > 0 }
func (n Name) IsEmbedded() bool { return n.embedded }
func (n Name) IsBlank() bool    { return n.Name() == `_` }
func (n Name) Name() string     { return n.name }
func (n Name) Tag() string      { return n.tag }

//gopherjs:purge Used for byte encoding of name, not used in JS
func writeVarint(buf []byte, n int) int

//gopherjs:purge Used for byte encoding of name, not used in JS
func (n Name) DataChecked(off int, whySafe string) *byte

//gopherjs:purge Used for byte encoding of name, not used in JS
func (n Name) Data(off int) *byte

//gopherjs:purge Used for byte encoding of name, not used in JS
func (n Name) ReadVarint(off int) (int, int)

func NewName(n, tag string, exported, embedded bool) Name {
	return Name{
		name:     n,
		tag:      tag,
		exported: exported,
		embedded: embedded,
	}
}
