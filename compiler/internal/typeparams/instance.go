package typeparams

import (
	"fmt"
	"go/types"

	"github.com/gopherjs/gopherjs/compiler/internal/symbol"
	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

// Instance of a generic type or function.
//
// Non-generic objects can be represented as an Instance with zero type params,
// they are instances of themselves.
type Instance struct {
	Object types.Object       // Object to be instantiated.
	TArgs  typesutil.TypeList // Type params to instantiate with.
}

// String returns a string representation of the Instance.
//
// Two semantically different instances may have the same string representation
// if the instantiated object or its type arguments shadow other types.
func (i *Instance) String() string {
	sym := symbol.New(i.Object).String()
	if len(i.TArgs) == 0 {
		return sym
	}

	return fmt.Sprintf("%s<%s>", sym, i.TArgs)
}

// TypeString returns a Go type string representing the instance (suitable for %T verb).
func (i *Instance) TypeString() string {
	tArgs := ""
	if len(i.TArgs) > 0 {
		tArgs = "[" + i.TArgs.String() + "]"
	}
	return fmt.Sprintf("%s.%s%s", i.Object.Pkg().Name(), i.Object.Name(), tArgs)
}

// IsTrivial returns true if this is an instance of a non-generic object.
func (i *Instance) IsTrivial() bool {
	return len(i.TArgs) == 0
}

// Recv returns an instance of the receiver type of a method.
//
// Returns zero value if not a method.
func (i *Instance) Recv() Instance {
	sig, ok := i.Object.Type().(*types.Signature)
	if !ok {
		return Instance{}
	}
	recv := typesutil.RecvType(sig)
	if recv == nil {
		return Instance{}
	}
	return Instance{
		Object: recv.Obj(),
		TArgs:  i.TArgs,
	}
}

// InstanceSet allows collecting and processing unique Instances.
//
// Each Instance may be added to the set any number of times, but it will be
// returned for processing exactly once. Processing order is not specified.
type InstanceSet struct {
	values      []Instance
	unprocessed int              // Index in values for the next unprocessed element.
	seen        InstanceMap[int] // Maps instance to a unique numeric id.
}

// Add instances to the set. Instances that have been previously added to the
// set won't be requeued for processing regardless of whether they have been
// processed already.
func (iset *InstanceSet) Add(instances ...Instance) *InstanceSet {
	for _, inst := range instances {
		if iset.seen.Has(inst) {
			continue
		}
		iset.seen.Set(inst, iset.seen.Len())
		iset.values = append(iset.values, inst)
	}
	return iset
}

// ID returns a unique numeric identifier assigned to an instance in the set.
// The ID is guaranteed to be unique among all instances of the same object
// within a given program. The ID will be consistent, as long as instances are
// added to the set in the same order.
//
// In order to have an ID assigned, the instance must have been previously added
// to the set.
//
// Note: we these ids are used in the generated code as keys to the specific
// type/function instantiation in the type/function object. Using this has two
// advantages:
//
// - More compact generated code compared to string keys derived from type args.
//
// - Collision avoidance in case of two different types having the same name due
// to shadowing.
//
// Here's an example where it's very difficult to assign non-colliding
// name-based keys to the two different types T:
//
//	func foo() {
//	    type T int
//	    { type T string } // Code block creates a new nested scope allowing for shadowing.
//	}
func (iset *InstanceSet) ID(inst Instance) int {
	id, ok := iset.seen.get(inst)
	if !ok {
		panic(fmt.Errorf("requesting ID of instance %v that hasn't been added to the set", inst))
	}
	return id
}

// next returns the next Instance to be processed.
//
// If there are no unprocessed instances, the second returned value will be false.
func (iset *InstanceSet) next() (Instance, bool) {
	if iset.exhausted() {
		return Instance{}, false
	}
	next := iset.values[iset.unprocessed]
	iset.unprocessed++
	return next, true
}

// exhausted returns true if there are no unprocessed instances in the set.
func (iset *InstanceSet) exhausted() bool { return len(iset.values) <= iset.unprocessed }

// Values returns instances that are currently in the set. Order is not specified.
func (iset *InstanceSet) Values() []Instance {
	return iset.values
}
