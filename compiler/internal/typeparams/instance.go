package typeparams

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/gopherjs/gopherjs/compiler/internal/symbol"
	"golang.org/x/exp/maps"
)

// Instance of a generic type or function.
//
// Non-generic objects can be represented as an Instance with zero type params,
// they are instances of themselves.
type Instance struct {
	Object types.Object // Object to be instantiated.
	TArgs  []types.Type // Type params to instantiate with.
}

// ID returns a string that uniquely identifies an instantiation of the generic
// object with the provided type arguments.
func (i *Instance) ID() string {
	sym := symbol.New(i.Object).String()
	if len(i.TArgs) == 0 {
		return sym
	}

	return fmt.Sprintf("%s<%s>", sym, i.Key())
}

// Key returns a string that uniquely identifies this instance among other
// instances of this particular object.
//
// Although in practice it is derived from type arguments, no particular
// guarantees are made about format of content of the string.
func (i *Instance) Key() string {
	buf := strings.Builder{}
	for i, tArg := range i.TArgs {
		if i != 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(types.TypeString(tArg, nil))
	}
	return buf.String()
}

// IsTrivial returns true if this is an instance of a non-generic object.
func (i *Instance) IsTrivial() bool {
	return len(i.TArgs) == 0
}

// InstanceSet allows collecting and processing unique Instances.
//
// Each Instance may be added to the set any number of times, but it will be
// returned for processing exactly once. Processing order is not specified.
type InstanceSet struct {
	unprocessed []Instance
	seen        map[string]Instance
}

// Add instances to the set. Instances that have been previously added to the
// set won't be requeued for processing regardless of whether they have been
// processed already.
func (iset *InstanceSet) Add(instances ...Instance) *InstanceSet {
	for _, inst := range instances {
		if _, ok := iset.seen[inst.ID()]; ok {
			continue
		}
		if iset.seen == nil {
			iset.seen = map[string]Instance{}
		}
		iset.unprocessed = append(iset.unprocessed, inst)

		iset.seen[inst.ID()] = inst
	}
	return iset
}

// next returns the next Instance to be processed.
//
// If there are no unprocessed instances, the second returned value will be false.
func (iset *InstanceSet) next() (Instance, bool) {
	if iset.exhausted() {
		return Instance{}, false
	}
	idx := len(iset.unprocessed) - 1
	next := iset.unprocessed[idx]
	iset.unprocessed = iset.unprocessed[0:idx]
	return next, true
}

// exhausted returns true if there are no unprocessed instances in the set.
func (iset *InstanceSet) exhausted() bool { return len(iset.unprocessed) == 0 }

// Values returns instances that are currently in the set. Order is not specified.
func (iset *InstanceSet) Values() []Instance {
	return maps.Values(iset.seen)
}
