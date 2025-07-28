package typesutil

import (
	"go/types"
	"strings"
)

// TypeList an ordered list of types.
type TypeList []types.Type

func (tl TypeList) String() string {
	return tl.TypesString(nil)
}

func (tl TypeList) TypesString(qt types.Qualifier) string {
	buf := strings.Builder{}
	for i, typ := range tl {
		if i != 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(types.TypeString(typ, qt))
	}
	return buf.String()
}

// Equal returns true if both lists of type arguments are identical.
func (tl TypeList) Equal(other TypeList) bool {
	if len(tl) != len(other) {
		return false
	}
	for i := range tl {
		if !types.Identical(tl[i], other[i]) {
			return false
		}
	}
	return true
}
