//go:build js
// +build js

package reflectlite_test

import (
	"testing"

	. "internal/reflectlite"
)

func TestTypes(t *testing.T) {
	for i, tt := range typeTests {
		if i == 30 {
			continue
		}
		testReflectType(t, i, Field(ValueOf(tt.i), 0).Type(), tt.s)
	}
}

func TestNameBytesAreAligned(t *testing.T) {
	t.Skip("TestNameBytesAreAligned")
}

// `A` is used with `B[T any]` and is otherwise not needed.
//
//gopherjs:purge for go1.19 without generics
type A struct{}

//gopherjs:purge for go1.19 without generics
type B[T any] struct{}

// removing the name tests using `B[T any]` for go1.19 without generics
var nameTests = []nameTest{
	{(*int32)(nil), "int32"},
	{(*D1)(nil), "D1"},
	{(*[]D1)(nil), ""},
	{(*chan D1)(nil), ""},
	{(*func() D1)(nil), ""},
	{(*<-chan D1)(nil), ""},
	{(*chan<- D1)(nil), ""},
	{(*any)(nil), ""},
	{(*interface {
		F()
	})(nil), ""},
	{(*TheNameOfThisTypeIsExactly255BytesLongSoWhenTheCompilerPrependsTheReflectTestPackageNameAndExtraStarTheLinkerRuntimeAndReflectPackagesWillHaveToCorrectlyDecodeTheSecondLengthByte0123456789_0123456789_0123456789_0123456789_0123456789_012345678)(nil), "TheNameOfThisTypeIsExactly255BytesLongSoWhenTheCompilerPrependsTheReflectTestPackageNameAndExtraStarTheLinkerRuntimeAndReflectPackagesWillHaveToCorrectlyDecodeTheSecondLengthByte0123456789_0123456789_0123456789_0123456789_0123456789_012345678"},
}
