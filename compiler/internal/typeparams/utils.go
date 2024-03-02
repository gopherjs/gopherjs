package typeparams

import "go/types"

// SignatureTypeParams returns receiver type params for methods, or function
// type params for standalone functions, or nil for non-generic functions and
// methods.
func SignatureTypeParams(sig *types.Signature) *types.TypeParamList {
	if tp := sig.RecvTypeParams(); tp != nil {
		return tp
	} else if tp := sig.TypeParams(); tp != nil {
		return tp
	} else {
		return nil
	}
}
