package typeparams

import (
	"fmt"
	"go/types"
	"sort"
	"strings"

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

	// TNest is the type params of the function this object was nested with-in.
	// e.g. In `func A[X any]() { type B[Y any] struct {} }` the `X`
	// from `A` is the context of `B[Y]` thus creating `B[X;Y]`.
	TNest typesutil.TypeList
}

// String returns a string representation of the Instance.
//
// Two semantically different instances may have the same string representation
// if the instantiated object or its type arguments shadow other types.
func (i Instance) String() string {
	return i.symbolicName() + i.TypeParamsString(`<`, `>`)
}

// TypeString returns a Go type string representing the instance (suitable for %T verb).
func (i Instance) TypeString() string {
	return i.qualifiedName() + i.TypeParamsString(`[`, `]`)
}

// symbolicName returns a string representation of the instance's name
// including the package name and pointer indicators but
// excluding the type parameters.
func (i Instance) symbolicName() string {
	if i.Object == nil {
		return `<nil>`
	}
	return symbol.New(i.Object).String()
}

// qualifiedName returns a string representation of the instance's name
// including the package name but
// excluding the type parameters and pointer indicators.
func (i Instance) qualifiedName() string {
	if i.Object == nil {
		return `<nil>`
	}
	if i.Object.Pkg() == nil {
		return i.Object.Name()
	}
	return fmt.Sprintf("%s.%s", i.Object.Pkg().Name(), i.Object.Name())
}

// TypeParamsString returns part of a Go type string that represents the type
// parameters of the instance including the nesting type parameters, e.g. [X;Y,Z].
func (i Instance) TypeParamsString(open, close string) string {
	hasNest := len(i.TNest) > 0
	hasArgs := len(i.TArgs) > 0
	buf := strings.Builder{}
	if hasNest || hasArgs {
		buf.WriteString(open)
		if hasNest {
			buf.WriteString(i.TNest.String())
			buf.WriteRune(';')
			if hasArgs {
				buf.WriteRune(' ')
			}
		}
		if hasArgs {
			buf.WriteString(i.TArgs.String())
		}
		buf.WriteString(close)
	}
	return buf.String()
}

// IsTrivial returns true if this is an instance of a non-generic object
// and it is not nested in a generic function.
func (i Instance) IsTrivial() bool {
	return len(i.TArgs) == 0 && len(i.TNest) == 0
}

// Recv returns an instance of the receiver type of a method.
//
// Returns zero value if not a method.
func (i Instance) Recv() Instance {
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
// Note: these ids are used in the generated code as keys to the specific
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

// ByObj returns instances grouped by object they belong to. Order is not specified.
func (iset *InstanceSet) ByObj() map[types.Object][]Instance {
	result := map[types.Object][]Instance{}
	for _, inst := range iset.values {
		result[inst.Object] = append(result[inst.Object], inst)
	}
	return result
}

// ForObj returns the instances that belong to the given object type.
// Order is not specified. This returns the same values as `ByObj()[obj]`.
func (iset *InstanceSet) ForObj(obj types.Object) []Instance {
	result := []Instance{}
	for _, inst := range iset.values {
		if inst.Object == obj {
			result = append(result, inst)
		}
	}
	return result
}

// ObjHasInstances returns true if there are any instances (either trivial
// or non-trivial) that belong to the given object type, otherwise false.
func (iset *InstanceSet) ObjHasInstances(obj types.Object) bool {
	for _, inst := range iset.values {
		if inst.Object == obj {
			return true
		}
	}
	return false
}

// PackageInstanceSets stores an InstanceSet for each package in a program, keyed
// by import path.
type PackageInstanceSets map[string]*InstanceSet

// Pkg returns InstanceSet for objects defined in the given package.
func (i PackageInstanceSets) Pkg(pkg *types.Package) *InstanceSet {
	path := pkg.Path()
	iset, ok := i[path]
	if !ok {
		iset = &InstanceSet{}
		i[path] = iset
	}
	return iset
}

// Add instances to the appropriate package's set. Automatically initialized
// new per-package sets upon a first encounter.
func (i PackageInstanceSets) Add(instances ...Instance) {
	for _, inst := range instances {
		i.Pkg(inst.Object.Pkg()).Add(inst)
	}
}

// ID returns a unique numeric identifier assigned to an instance in the set.
//
// See: InstanceSet.ID().
func (i PackageInstanceSets) ID(inst Instance) int {
	return i.Pkg(inst.Object.Pkg()).ID(inst)
}

func (i PackageInstanceSets) String() string {
	pkgName := make([]string, 0, len(i))
	for pkg := range i {
		pkgName = append(pkgName, pkg)
	}
	sort.Strings(pkgName)
	buf := strings.Builder{}
	for _, pkg := range pkgName {
		buf.WriteString(pkg)
		buf.WriteString(":\n")
		iset := i[pkg]
		for _, inst := range iset.values {
			buf.WriteString("\t")
			buf.WriteString(inst.String())
			buf.WriteString("\n")
		}
	}
	return buf.String()
}
