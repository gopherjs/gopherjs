# Dead-Code Elimination

Dead-Code Eliminations (DCE) is used to remove code that isn't
reachable from a code entry point. Entry points are code like the main method,
init functions, and variable initializations with side effects.
These entry points are always considered alive. Any dependency of
something alive, is also considered alive.

Once all dependencies are taken into consideration we have the set of alive
declarations. Anything not considered alive is considered dead and
may be safely eliminated, i.e. not outputted to the JS file(s).

- [Idea](#idea)
  - [Package](#package)
  - [Named Types](#named-types)
    - [Named Structs](#named-structs)
  - [Interfaces](#interfaces)
  - [Functions](#functions)
  - [Variables](#variables)
  - [Generics and Instances](#generics-and-instances)
  - [Links](#links)
- [Design](#design)
  - [Initially Alive](#initially-alive)
  - [Naming](#naming)
    - [Name Specifics](#name-specifics)
  - [Dependencies](#dependencies)
- [Examples](#examples)
  - [Dead Package](#dead-package)
  - [Grandmas and Zombies](#grandmas-and-zombies)
  - [Side Effects](#side-effects)
  - [Instance Duck-typing](#instance-duck-typing)
- [Additional Notes](#additional-notes)

## Idea

The following is the logic behind the DCE mechanism. Not all of the following
is used since some conditions are difficult to determine even with a lot of
additional information, and because GopherJS stores some additional information
making some parts of DCE unnecessary. To ensure that the JS output is fully
functional, we bias the DCE towards things being alive. We'd rather keep
something we don't need than remove something that is needed.

### Package

Package declarations (e.g. `package foo`) might be able to be removed
when only used by dead-code. However, packages may be imported and not used
for various reasons including to invoke some initialization or to implement
a link. So it is difficult to determine.
See [Dead Package](#dead-package) example.

Currently, we won't remove any packages, but someday the complexity
could be added to check for inits, side effects, links, etc then determine
if any of those are are alive or affect alive things.

### Named Types

Named type definitions (e.g. `type Foo int`) depend on
the underlying type for each definition.

When a named type is alive, all of its exported methods
(e.g. `func (f Foo) Bar() { }`) are also alive, even any unused exported method.
Unused exported methods are still important when duck-typing.
See [Interfaces](#interfaces) for more information.
See [Grandmas and Zombies](#grandmas-and-zombies) for an example of what
can happen when removing an unused exported method.

Also unused exported methods could be accessed by name via reflect
(e.g. `reflect.ValueOf(&Foo{}).MethodByName("Bar")`). Since the
string name may be provided from outside the code, such as the command line,
it is impossible to determine which exported methods could be accessed this way.
It would be very difficult to determine which types are ever accessed via
reflect so by default we simply assume any can be.

Methods that are unexported may be considered dead when unused even when
the receiver type is alive. The exception is when an interface in the same
package has the same unexported method in it.
See [Interfaces](#interfaces) for more information.

#### Named Structs

A named struct is a named type that has a struct as its underlying type,
e.g. `type Foo struct { }`. A struct type depends on all of the types in
its fields and embedded fields.

If the struct type is alive then all the types of the fields will also be alive.
Even unexported fields maybe accessed via reflections, so they all must be
alive. Also, the fields are needed for comparisons and serializations
(such as `encoding/binary`).

### Interfaces

All the types in the function signatures and embedded interfaces are the
dependents of the interface.

Interfaces may contain exported and unexported function signatures.
If an interface is alive then all of the functions are alive.
Since there are many ways to wrap a type with an interface, any alive type that
duck-types to an interface must have all of the matching methods also alive.

In theory the unexported functions are also alive however, for GopherJS there
is an exception because duck-typing is handled separately from the method
definitions. Those difference are discussed in [Dependencies](#dependencies)
but for this idea we discuss DCE more generally.

Since the exported methods in an alive type will be alive, see
[Named Types](#named-types), the only ones here that need to be considered
are the unexported methods. An interface with unexported methods may only
duck-type to types within the package the interface is defined in.
Therefore, if an interface is alive with unexported methods, then all
alive types within the same package that duck-type to that interface,
will have the matching unexported methods be alive.

Since doing a full `types.Implements` check between every named types and
interfaces in a package is difficult, we simplify this requirement to be
any unexported method in an alive named type that matches an unexported
method in an alive interface is alive even if the named type doesn't duck-type
to the interface. This means that in some rare cases, some unexported
methods on named structs that could have been eliminated will not be.
For example, given `type Foo struct{}; func(f Foo) X(); func (f Foo) y()` the
`Foo.y()` method may be alive if `types Bar interface { Z(); y() }` is alive
even though the `X()` and `Z()` means that `Foo` doesn't implement `Bar`
and therefore `Foo.y()` can not be called via a `Bar.y()`.

We will try to reduce the false positives in alive unexported methods by using
the parameter and result types of the methods. Meaning that
 `y()`, `y(int)`, `y() int`, etc won't match just because they are named `y`.

### Functions

Functions with or without a receiver are dependent on the types used by the
parameters, results, and type uses inside the body of the function.
They are also dependent on any function invoked or used, and
any package level variable that is used.

Unused functions without a receiver, that are exported or not, may be
considered dead since they aren't used in duck-typing and cannot be accessed
by name via reflections.

### Variables

Variables (or constants) depend on their type and anything used during
initialization.

The exported or unexported variables are dead unless they are used by something
else that is alive or if the initialization has side effects.

If the initialization has side effects the variable will be alive even
if unused. The side effect may be simply setting another variable's value
that is also unused, however it would be difficult to determine if the
side effects are used or not.
See [Side Effects](#side-effects) example.

### Generics and Instances

For functions and types with generics, the definitions are split into
unique instances. For example, `type StringKeys[T any] map[string]T`
could be used in code as `StringKeys[int]` and `StringKeys[*Cat]`.
We don't need all possible instances, only the ones which are realized
in code. Each instance depends on the realized parameter types (instance types).
In the example the instance types are `int` and `*Cat`.

The instance of the generic type also defines the code with the specific
instance types (e.g. `map[string]int` and `map[string]*Cat`). When an
instance is depended on by alive code, only that instance is alive, not the
entire generic type. This means if `StringKey[*Cat]` is only used from dead
code, it is also dead and can be safely eliminated.

The named generic types may have methods that are also copied for an instance
with the parameter types replaced by the instance types. For example,
`func (sk StringKeys[T]) values() []T { ... }` becomes
`func (sk StringKeys[int]) values() []int { ... }` when the instance type
is `int`. This method in the instance now duck-types to
`interface { values() []int }` and therefore must follow the rules for
unexported methods.
See [Instance Duck-typing](#instance-duck-typing) example for more information.

Functions and named types may be generic, but methods and unnamed types
may not be. This makes somethings simpler. A method with a receiver is used,
only the receiver's instance types are needed. The generic type or function
may not be needed since only the instances are written out.

This also means that inside of a generic function or named type there is only
one type parameter list being used. Even generic types used inside of the
generic function must be specified in terms of the type parameter for the
generic and doesn't contribute any type parameters of it's own.
For example, inside of `func Foo[K comparable, V any]() { ... }` every
usage of a generic type must specify a concrete type (`int`, `*Cat`,
`Bar[Bar[bool]]`) or use the parameter types `K` and `V`. This is simpler
than languages that allow a method of an object to have it's own type
parameters, e.g. `class X<T> { void Y<U>() { ... } ... }`.

However, generics mean that the same method, receiver, type, etc names
will be used with different parameters types caused by different instance
types. The instance types are the type arguments being passed into those
parameter types for a specific instance.
When an interface is alive, the signatures for unexported methods
need to be instantiated with type arguments so that we know which instances
the interface is duck-typing to.

### Links

Links use compiler directives
([`//go:linkname`](https://pkg.go.dev/cmd/compile#hdr-Compiler_Directives))
to alias a `var` or `func` with another.
For example some code may have `func bar_foo()` as a function stub that is
linked with `foo() { ... }` as a function with a body, i.e. the target of the
link. The links are single directional but allow multiple stubs to link to the
same target.

When a link is made, the dependencies for the linked code come from
the target. If the target is used by something alive then it is alive.
If a stub linked to a target is used by something alive then that stub and
the target are both alive.

Since links cross package boundaries in ways that may violate encapsulation
and the dependency tree, it may be difficult to determine if a link is alive
or not. Therefore, currently all links are considered alive.

## Design

The design is created taking all the parts of the above idea together and
simplifying the justifications down to a simple set of rules.

### Initially alive

- The `main` method in the `main` package
- The `init` in every included file
- Any variable initialization that has a side effect
- Any linked function or variable
- Anything not named

### Naming

The following specifies what declarations should be named and how
the names should look. These names are later used to match (via string
comparisons) dependencies with declarations that should be set as alive.
Since the names are used to filter out alive code from all the code
these names may also be referred to as filters.

Some names will have multiple name parts; an object name and method name.
This is kind of like a first name and last name when a first name alone isn't
specific enough. This helps with matching multiple dependency requirements
for a declaration, i.e. both name parts must be alive before the declaration
is considered alive.

Currently, only unexported method declarations will have a method
name to support duck-typing with unexported signatures on interfaces.
If the unexported method is depended on, then both names will be in
the dependencies. If the receiver is alive and an alive interface has the
matching unexported signature, then both names will be depended on thus making
the unexported method alive. Since the unexported method is only visible in
the package in which it is defined, the package path is included in the
method name.

To simplify the above for GopherJS, we don't look at the receiver for
an unexported method before indicating it is alive. Meaning if there is no
interface, only two named objects with identical unexported methods, the use
of either will indicate a use of both. This will cause slightly more unexported
method to be alive while reducing the complication of type checking which object
or type of object is performing the call.

| Declaration | exported | unexported | non-generic | generic | object name | method name |
|:------------|:--------:|:----------:|:-----------:|:-------:|:------------|:------------|
| variables  | █ | █ | █ | n/a | `<package>.<var name>` | |
| functions  | █ | █ | █ |     | `<package>.<func name>` | |
| functions  | █ | █ |   |  █  | `<package>.<func name>[<type args>]` | |
| named type | █ | █ | █ |     | `<package>.<type name>` | |
| named type | █ | █ |   |  █  | `<package>.<type name>[<type args>]` | |
| method     | █ |   | █ |     | `<package>.<receiver name>` | |
| method     | █ |   |   |  █  | `<package>.<receiver name>[<type args>]` | |
| method     |   | █ | █ |     | `<package>.<receiver name>` | `<package>.<method name>(<parameter types>)(<result types>)` |
| method     |   | █ |   |  █  | `<package>.<receiver name>[<type args>]` | `<package>.<method name>(<parameter types>)(<result types>)` |

#### Name Specifics

The following are specifics about the different types of names that show
up in the above table. This isn't the only way to represent this information.
These names can get long but don't have to. The goal is to make the names
as unique as possible whilst still ensuring that signatures in
interfaces will still match the correct methods. The less unique
the more false positives for alive will occur meaning more dead code is
kept alive. However, too unique could cause needed alive code to not match
and be eliminated causing the application to not run.

`<package>.<var name>`, `<package>.<func name>`, `<package>.<type name>`
and `<package>.<receiver name>` all have the same form. They are
the package path, if there is one, followed by a `.` and the object name
or receiver name. For example [`rand.Shuffle`](https://pkg.go.dev/math/rand@go1.23.1#Shuffle)
will be named `math/rand.Shuffle`. The builtin [`error`](https://pkg.go.dev/builtin@go1.23.1#error)
will be named `error` without a package path.

`<package>.<func name>[<type args>]`, `<package>.<type name>[<type args>]`,
and `<package>.<receiver name>[<type args>]` are the same as above
except with comma separated type arguments in square brackets.
The type arguments are either the instance types, or type parameters
since the instance type could be a match for the type parameter on the
generic. For example `type Foo[T any] struct{}; type Bar[B any] { f Foo[B] }`
has `Foo[B]` used in `Bar` that is identical to `Foo[T]` even though
technically `Foo[B]` is an instance of `Foo[T]` with `B` as the type argument.

Command compiles, i.e. compiles with a `main` entry point, and test builds
should not have any instance types that aren't resolved to concrete types,
however to handle partial compiles of packages, instance types may still
be a type parameter, including unions of approximate constraints,
i.e. `~int|~string`.

Therefore, type arguments need to be reduced to only types. This means
something like [`maps.Keys`](https://pkg.go.dev/maps@go1.23.1#Keys), i.e.
`func Keys[Map ~map[K]V, K comparable, V any](m Map) iter.Seq[K]`,
will be named `maps.Keys[~map[comparable]any, comparable, any]` as a generic.
If the instances for `Map` are `map[string]int` and `map[int][]*cats.Cat`,
then respectively the names would be `maps.Keys[map[string]int, string, int]`
and `maps.Keys[map[int][]*cats.Cat, int, []*cats.Cat]`. If this function is used
in `func Foo[T ~string|~int](data map[string]T) { ... maps.Keys(data) ... }`
then the instance of `maps.Keys` that `Foo` depends on would be named
`maps.Keys[map[string]~int|~string, string, ~int|~string]`.

For the method name of unexposed methods,
`<package>.<method name>(<parameter types>)(<result types>)`, the prefix,
`<package>.<method name>`, is in the same format as `<package>.<func name>`.
The rest contains the signature, `(<parameter types>)(<result types>)`.
The signature is defined with only the types since
`(v, u int)(ok bool, err error)` should match `(x, y int)(bool, error)`.
To match both will have to be `(int, int)(bool, error)`.
Also the parameter types should include the veridic indicator,
e.g. `sum(...int)int`, since that affects how the signature is matched.
If there are no results then the results part is left off. Otherwise,
the result types only need parenthesis if there are more than one result,
e.g. `(int, int)`, `(int, int)bool`, and `(int, int)(bool, error)`.

In either the object name or method name, if there is a recursive
type parameter, e.g. `func Foo[T Bar[T]]()` the second usage of the
type parameter will have it's type parameters as `...` to prevent an
infinite loop whilst also indicating which object in the type parameter
is recursive, e.g. `Foo[Bar[Bar[...]]]`.

### Dependencies

The dependencies that are specified in an expression.
For example a function that invokes another function will be dependent on
that invoked function. When a dependency is added it will be added as one
or more names to the declaration that depends on it. It follows the
[naming rules](#naming) so that the dependencies will match correctly.

In theory, structural dependencies would be needed to be added
automatically while the declaration is being named. When an interface is named,
it would automatically add all unexported signatures as dependencies via
`<package path>.<method name>(<parameter type list>)(<result type list>)`.
However, we do not need to do that in GopherJS because we aren't using
the existence of realized methods in duck-typing. GopherJS stores full set
of method information when describing the type so that even when things like
unexported methods in interfaces are removed, duck-typing will still work
correctly. This reduces the size of the code by not keeping a potentially
long method body when the signature is all that is needed.

Currently we don't filter unused packages so there is no need to automatically
add dependencies on the packages themselves. This is also why the package
declarations aren't named and therefore are always alive.

## Examples

### Dead Package

In this example, a point package defines a `Point` object.
The point package may be used by several repos as shared code so can not
have code manually removed from it to reduce its dependencies for specific
applications.

For the current example, the `Distance` method is never used and therefore
dead. The `Distance` method is the only method dependent on the math package.
It might be safe to make the whole math package dead too and eliminate it in
this case, however, it is possible that some packages aren't used on purpose
and their reason for being included is to invoke the initialization functions
within the package. If a package has any inits or any variable definitions
with side effects, then the package can not be safely removed.

```go
package point

import "math"

type Point struct {
   X float64
   Y float64
}

func (p Point) Sub(other Point) Point {
   p.X -= other.X
   p.Y -= other.Y
   return p
}

func (p Point) ToQuadrant1() Point {
   if p.X < 0.0 {
      p.X = -p.X
   }
   if p.Y < 0.0 {
      p.Y = -p.Y
   }
   return p
}

func (p Point) Manhattan(other Point) float64 {
   a := p.Sub(other).ToQuadrant1()
   return a.X + a.Y
}

func (p Point) Distance(other Point) float64 {
   d := p.Sub(other)
   return math.Sqrt(d.X*d.X + d.Y*d.Y)
}
```

```go
package main

import "point"

func main() {
   a := point.Point{X: 10.2, Y: 45.3}
   b := point.Point{X: -23.0, Y: 7.7}
   println(`Manhatten a to b:`, a.Manhattan(b))
}
```

### Grandmas and Zombies

In this example, the following code sorts grandmas and zombies by if they are
`Dangerous`. The method `EatBrains` is never used. If we remove `EatBrains`
from `Zombie` then both the grandmas and zombies are moved to the safe
bunker. If we remove `EatBrains` from `Dangerous` then both grandmas and
zombies will be moved to the air lock because `Dangerous` will duck-type
to all `Person` instances. Unused exported methods and signatures must be
considered alive if the type is alive.

```go
package main

import "fmt"

type Person interface {
   MoveTo(loc string)
}

type Dangerous interface {
   Person
   EatBrains()
}

type Grandma struct{}

func (g Grandma) MoveTo(loc string) {
   fmt.Println(`grandma was moved to`, loc)
}

type Zombie struct{}

func (z Zombie) MoveTo(loc string) {
   fmt.Println(`zombie was moved to`, loc)
}

func (z Zombie) EatBrains() {}

func main() {
   people := []Person{Grandma{}, Zombie{}, Grandma{}, Zombie{}}
   for _, person := range people {
      if _, ok := person.(Dangerous); ok {
         person.MoveTo(`air lock`)
      } else {
         person.MoveTo(`safe bunker`)
      }
   }
}
```

### Side Effects

In this example unused variables are being initialized with expressions
that have side effects. The `max` value is 8 by the time `main` is called
because each initialization calls `count()` that increments `max`.
The expression doesn't have to have a function call and can be any combination
of operations.

An initialization may have a side effect even if it doesn't set a value. For
example, simply printing a message to the console is a side effect that
can not be removed even if it is part of an unused initializer.

```go
package main

import "fmt"

func count() int {
   max++
   return max
}

var (
   max  = 0
   _    = count() // a
   b, c = count(), count()
   x    = []int{count(), count(), count()}[0]
   y, z = func() (int, int) { return count(), count() }()
)

func main() {
   fmt.Println(`max count`, max) // Outputs: max count 8
}
```

### Instance Duck-typing

In this example the type `StringKeys[T any]` is a map that stores
any kind of value with string keys. There is an interface `IntProvider`
that `StringKeys` will duck-type to iff the instance type is `int`,
i.e. `StringKeys[int]`. This exemplifies how the instance types used
in the type arguments affect the overall signature such that in some
cases a generic object may match an interface and in others it may not.

Also notice that the structure was typed with `T` as the parameter type's
name whereas the methods use `S`. This shows that the name of the type
doesn't matter in the instancing. Therefore, outputting a methods name
(assuming it is unexported) should use the instance type not the parameter
name, e.g. `value() []int` or `value() []any` instead of `value() []S` or
`value() []T`.

```go
package main

import (
   "fmt"
   "sort"
)

type StringKeys[T any] map[string]T

func (sk StringKeys[S]) Keys() []string {
   keys := make([]string, 0, len(sk))
   for key := range sk {
      keys = append(keys, key)
   }
   sort.Strings(keys)
   return keys
}

func (sk StringKeys[S]) Values() []S {
   values := make([]S, len(sk))
   for i, key := range sk.Keys() {
      values[i] = sk[key]
   }
   return values
}

type IntProvider interface {
   Values() []int
}

func Sum(data IntProvider) int {
   sum := 0
   for _, value := range data.Values() {
      sum += value
   }
   return sum
}

func main() {
   sInt := StringKeys[int]{
      `one`:   1,
      `two`:   2,
      `three`: 3,
      `four`:  4,
   }
   fmt.Println(sInt.Keys())   // Outputs: [four one three two]
   fmt.Println(sInt.Values()) // Outputs: [4 1 3 2]
   fmt.Println(Sum(sInt))     // Outputs: 10

   sFp := StringKeys[float64]{
      `one`:   1.1,
      `two`:   2.2,
      `three`: 3.3,
      `four`:  4.4,
   }
   fmt.Println(sFp.Keys())   // Outputs: [four one three two]
   fmt.Println(sFp.Values()) // [4.4 1.1 3.3 2.2]
   //fmt.Println(Sum(sFp))   // Fails with "StringKeys[float64] does not implement IntProvider"
}
```

## Additional Notes

This DCE is different from those found in
Muchnick, Steven S.. “Advanced Compiler Design and Implementation.” (1997),
Chapter 18 Control-Flow and Low-Level Optimization,
Section 10 Dead-Code Elimination. And different from related DCE designs
such as Knoop, Rüthing, and Steffen. "Partial dead code elimination." (1994),
SIGPLAN Not. 29, 6, 147–158.
See [DCE wiki](https://en.wikipedia.org/wiki/Dead-code_elimination)
for more information.

Those discuss DCE at the block code level where the higher level
constructs such as functions and objects have been reduced to a graphs of
blocks with variables, procedures, and routines. Since we want to keep the
higher level constructs during transpilation, we simply are reducing
the higher level constructs not being used.

Any variable internal to the body of a function or method that is unused or
only used for computing new values for itself, are left as is.
The Go compiler and linters have requirements that attempt to prevent this
kind of dead-code in a function body (so long as an underscore isn't used to quite
usage warnings) and prevent unreachable code. Therefore, we aren't going to
worry about trying to DCE inside of function bodies or in variable initializers.

GopherJS does not implicitly perform JS Tree Shaking Algorithms, as discussed in
[How Modern Javascript eliminate dead code](https://blog.stackademic.com/how-modern-javascript-eliminates-dead-code-tree-shaking-algorithm-d7861e48df40)
(2023) at this time and provides no guarantees about the effectiveness
of running such an algorithm on the resulting JS.
