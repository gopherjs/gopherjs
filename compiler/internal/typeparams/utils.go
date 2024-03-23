package typeparams

import (
	"errors"
	"fmt"
	"go/types"
)

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

var (
	errInstantiatesGenerics = errors.New("instantiates generic type or function")
	errDefinesGenerics      = errors.New("defines generic type or function")
)

// RequiresGenericsSupport an error if the type-checked code depends on
// generics support.
func RequiresGenericsSupport(info *types.Info) error {
	type withTypeParams interface{ TypeParams() *types.TypeParamList }

	for ident := range info.Instances {
		// Any instantiation means dependency on generics.
		return fmt.Errorf("%w: %v", errInstantiatesGenerics, info.ObjectOf(ident))
	}

	for _, obj := range info.Defs {
		if obj == nil {
			continue
		}
		typ, ok := obj.Type().(withTypeParams)
		if ok && typ.TypeParams().Len() > 0 {
			return fmt.Errorf("%w: %v", errDefinesGenerics, obj)
		}
	}

	return nil
}
