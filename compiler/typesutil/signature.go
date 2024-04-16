package typesutil

import (
	"fmt"
	"go/types"
)

// Signature is a helper that provides convenient access to function
// signature type information.
type Signature struct {
	Sig *types.Signature
}

// RequiredParams returns the number of required parameters in the function signature.
func (st Signature) RequiredParams() int {
	l := st.Sig.Params().Len()
	if st.Sig.Variadic() {
		return l - 1 // Last parameter is a slice of variadic params.
	}
	return l
}

// VariadicType returns the slice-type corresponding to the signature's variadic
// parameter, or nil of the signature is not variadic. With the exception of
// the special-case `append([]byte{}, "string"...)`, the returned type is
// `*types.Slice` and `.Elem()` method can be used to get the type of individual
// arguments.
func (st Signature) VariadicType() types.Type {
	if !st.Sig.Variadic() {
		return nil
	}
	return st.Sig.Params().At(st.Sig.Params().Len() - 1).Type()
}

// Param returns the expected argument type for the i'th argument position.
//
// This function is able to return correct expected types for variadic calls
// both when ellipsis syntax (e.g. myFunc(requiredArg, optionalArgSlice...))
// is used and when optional args are passed individually.
//
// The returned types may differ from the actual argument expression types if
// there is an implicit type conversion involved (e.g. passing a struct into a
// function that expects an interface).
func (st Signature) Param(i int, ellipsis bool) types.Type {
	if i < st.RequiredParams() {
		return st.Sig.Params().At(i).Type()
	}
	if !st.Sig.Variadic() {
		// This should never happen if the code was type-checked successfully.
		panic(fmt.Errorf("tried to access parameter %d of a non-variadic signature %s", i, st.Sig))
	}
	if ellipsis {
		return st.VariadicType()
	}
	return st.VariadicType().(*types.Slice).Elem()
}

// HasResults returns true if the function signature returns something.
func (st Signature) HasResults() bool {
	return st.Sig.Results().Len() > 0
}

// HasNamedResults returns true if the function signature returns something and
// returned results are names (e.g. `func () (val int, err error)`).
func (st Signature) HasNamedResults() bool {
	return st.HasResults() && st.Sig.Results().At(0).Name() != ""
}
