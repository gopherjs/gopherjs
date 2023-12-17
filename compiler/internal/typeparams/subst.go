package typeparams

import (
	"go/token"
	"go/types"

	_ "unsafe" // For go:linkname.
)

type substMap map[*types.TypeParam]types.Type

//go:linkname goTypesCheckerSubst go/types.(*Checker).subst
func goTypesCheckerSubst(check *types.Checker, pos token.Pos, typ types.Type, smap substMap, ctxt *types.Context) types.Type
