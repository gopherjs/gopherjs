package typeparams

import "testing"

// This file contains test cases for low-level details of typeparam
// implementation like variable name assignment.

type TypeParamNameMismatch[T any] struct{}

func (TypeParamNameMismatch[T1]) M(_ T1) {}

func TestTypeParamNameMismatch(t *testing.T) {
	// This test case exercises the case when the same typeparam is named
	// differently between the struct definition and one of its methods. GopherJS
	// must allocate the same JS variable name to both instances of the type param
	// in order to make it possible for the reflection method data to be evaluated
	// within the type's generic factory function.

	a := TypeParamNameMismatch[int]{}
	a.M(0) // Make sure the method is not eliminated as dead code.
}

type (
	TypeParamVariableCollision1[T any] struct{}
	TypeParamVariableCollision2[T any] struct{}
	TypeParamVariableCollision3[T any] struct{}
)

func (TypeParamVariableCollision1[T]) M() {}
func (TypeParamVariableCollision2[T]) M() {}
func (TypeParamVariableCollision3[T]) M() {}

func TestTypeParamVariableCollision(t *testing.T) {
	// This test case exercises a situation when in minified mode the variable
	// name that gets assigned to the type parameter in the method's generic
	// factory function collides with a different variable in the type's generic
	// factory function. The bug occurred because the JS variable name allocated
	// to the *types.TypeName object behind a type param within the method's
	// factory function was not marked as used within type's factory function.

	// Note: to trigger the bug, a package should contain multiple generic types,
	// so that sequentially allocated minified variable names get far enough to
	// cause the collision.

	// Make sure types and methods are not eliminated as dead code.
	TypeParamVariableCollision1[int]{}.M()
	TypeParamVariableCollision2[int]{}.M()
	TypeParamVariableCollision3[int]{}.M()
}
