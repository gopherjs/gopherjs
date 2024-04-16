package typesutil

import (
	"go/token"
	"go/types"
	"testing"
)

func TestSignature_RequiredParams(t *testing.T) {
	tests := []struct {
		descr string
		sig   *types.Signature
		want  int
	}{{
		descr: "regular signature",
		sig: types.NewSignatureType(nil, nil, nil, types.NewTuple(
			types.NewVar(token.NoPos, nil, "a", types.Typ[types.Int]),
			types.NewVar(token.NoPos, nil, "b", types.Typ[types.String]),
			types.NewVar(token.NoPos, nil, "c", types.NewSlice(types.Typ[types.String])),
		), nil, false),
		want: 3,
	}, {
		descr: "variadic signature",
		sig: types.NewSignatureType(nil, nil, nil, types.NewTuple(
			types.NewVar(token.NoPos, nil, "a", types.Typ[types.Int]),
			types.NewVar(token.NoPos, nil, "b", types.Typ[types.String]),
			types.NewVar(token.NoPos, nil, "c", types.NewSlice(types.Typ[types.String])),
		), nil, true /*variadic*/),
		want: 2,
	}}

	for _, test := range tests {
		t.Run(test.descr, func(t *testing.T) {
			sig := Signature{Sig: test.sig}
			got := sig.RequiredParams()
			if got != test.want {
				t.Errorf("Got: {%s}.RequiredParams() = %d. Want: %d.", test.sig, got, test.want)
			}
		})
	}
}

func TestSignature_VariadicType(t *testing.T) {
	tests := []struct {
		descr string
		sig   *types.Signature
		want  types.Type
	}{{
		descr: "regular signature",
		sig: types.NewSignatureType(nil, nil, nil, types.NewTuple(
			types.NewVar(token.NoPos, nil, "a", types.Typ[types.Int]),
			types.NewVar(token.NoPos, nil, "b", types.Typ[types.String]),
			types.NewVar(token.NoPos, nil, "c", types.NewSlice(types.Typ[types.String])),
		), nil, false),
		want: nil,
	}, {
		descr: "variadic signature",
		sig: types.NewSignatureType(nil, nil, nil, types.NewTuple(
			types.NewVar(token.NoPos, nil, "a", types.Typ[types.Int]),
			types.NewVar(token.NoPos, nil, "b", types.Typ[types.String]),
			types.NewVar(token.NoPos, nil, "c", types.NewSlice(types.Typ[types.String])),
		), nil, true /*variadic*/),
		want: types.NewSlice(types.Typ[types.String]),
	}}

	for _, test := range tests {
		t.Run(test.descr, func(t *testing.T) {
			sig := Signature{Sig: test.sig}
			got := sig.VariadicType()
			if !types.Identical(got, test.want) {
				t.Errorf("Got: {%s}.VariadicType() = %v. Want: %v.", test.sig, got, test.want)
			}
		})
	}
}

func TestSignature_Param(t *testing.T) {
	sig := types.NewSignatureType(nil, nil, nil, types.NewTuple(
		types.NewVar(token.NoPos, nil, "a", types.Typ[types.Int]),
		types.NewVar(token.NoPos, nil, "b", types.Typ[types.Byte]),
		types.NewVar(token.NoPos, nil, "c", types.NewSlice(types.Typ[types.String])),
	), nil, true /*variadic*/)

	tests := []struct {
		descr    string
		param    int
		ellipsis bool
		want     types.Type
	}{{
		descr: "required param",
		param: 1,
		want:  types.Typ[types.Byte],
	}, {
		descr: "variadic param",
		param: 2,
		want:  types.Typ[types.String],
	}, {
		descr: "variadic param repeated",
		param: 3,
		want:  types.Typ[types.String],
	}, {
		descr:    "variadic param with ellipsis",
		param:    2,
		ellipsis: true,
		want:     types.NewSlice(types.Typ[types.String]),
	}}

	for _, test := range tests {
		t.Run(test.descr, func(t *testing.T) {
			sig := Signature{Sig: sig}
			got := sig.Param(test.param, test.ellipsis)
			if !types.Identical(got, test.want) {
				t.Errorf("Got: {%s}.Param(%v, %v) = %v. Want: %v.", sig, test.param, test.ellipsis, got, test.want)
			}
		})
	}
}

func TestSignature_HasXResults(t *testing.T) {
	tests := []struct {
		descr           string
		sig             *types.Signature
		hasResults      bool
		hasNamedResults bool
	}{{
		descr:           "no results",
		sig:             types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(), false),
		hasResults:      false,
		hasNamedResults: false,
	}, {
		descr: "anonymous result",
		sig: types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(
			types.NewVar(token.NoPos, nil, "", types.Typ[types.String]),
		), false),
		hasResults:      true,
		hasNamedResults: false,
	}, {
		descr: "named result",
		sig: types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(
			types.NewVar(token.NoPos, nil, "s", types.Typ[types.String]),
		), false),
		hasResults:      true,
		hasNamedResults: true,
	}, {
		descr: "underscore named result",
		sig: types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(
			types.NewVar(token.NoPos, nil, "_", types.Typ[types.String]),
		), false),
		hasResults:      true,
		hasNamedResults: true,
	}}

	for _, test := range tests {
		t.Run(test.descr, func(t *testing.T) {
			sig := Signature{Sig: test.sig}
			gotHasResults := sig.HasResults()
			if gotHasResults != test.hasResults {
				t.Errorf("Got: {%s}.HasResults() = %v. Want: %v.", test.sig, gotHasResults, test.hasResults)
			}
			gotHasNamedResults := sig.HasNamedResults()
			if gotHasNamedResults != test.hasNamedResults {
				t.Errorf("Got: {%s}.HasResults() = %v. Want: %v.", test.sig, gotHasNamedResults, test.hasNamedResults)
			}
		})
	}
}
