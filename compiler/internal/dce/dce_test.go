package dce

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"regexp"
	"sort"
	"testing"

	"github.com/gopherjs/gopherjs/compiler/typesutil"
)

func Test_Collector_CalledOnce(t *testing.T) {
	var c Collector
	decl1 := &testDecl{}
	decl2 := &testDecl{}

	err := capturePanic(t, func() {
		c.CollectDCEDeps(decl1, func() {
			c.CollectDCEDeps(decl2, func() {
				t.Fatal(`the nested collect function was called`)
			})
		})
	})
	errorMatches(t, err, `^called CollectDCEDeps inside another`)
}

func Test_Collector_Collecting(t *testing.T) {
	pkg := testPackage(`tristan`)
	obj1 := quickVar(pkg, `Primus`)
	obj2 := quickVar(pkg, `Secundus`)
	obj3 := quickVar(pkg, `Tertius`)
	obj4 := quickVar(pkg, `Quartus`)
	obj5 := quickVar(pkg, `Quintus`)
	obj6 := quickVar(pkg, `Sextus`)
	obj7 := quickVar(pkg, `Una`)

	decl1 := quickTestDecl(obj1)
	decl2 := quickTestDecl(obj2)
	var c Collector

	c.DeclareDCEDep(obj1) // no effect since a collection isn't running.
	depCount(t, decl1, 0)
	depCount(t, decl2, 0)

	c.CollectDCEDeps(decl1, func() {
		c.DeclareDCEDep(obj2)
		c.DeclareDCEDep(obj3)
		c.DeclareDCEDep(obj3) // already added so has no effect.
	})
	depCount(t, decl1, 2)
	depCount(t, decl2, 0)

	c.DeclareDCEDep(obj4) // no effect since a collection isn't running.
	depCount(t, decl1, 2)
	depCount(t, decl2, 0)

	c.CollectDCEDeps(decl2, func() {
		c.DeclareDCEDep(obj5)
		c.DeclareDCEDep(obj6)
		c.DeclareDCEDep(obj7)
	})
	depCount(t, decl1, 2)
	depCount(t, decl2, 3)

	// The second collection adds to existing dependencies.
	c.CollectDCEDeps(decl2, func() {
		c.DeclareDCEDep(obj4)
		c.DeclareDCEDep(obj5)
	})
	depCount(t, decl1, 2)
	depCount(t, decl2, 4)
}

func Test_Info_SetNameAndDep(t *testing.T) {
	tests := []struct {
		name string
		obj  types.Object
		want Info // expected Info after SetName
	}{
		{
			name: `package`,
			obj: parseObject(t, `Sarah`,
				`package jim
				import Sarah "fmt"`),
			want: Info{
				objectFilter: `jim.Sarah`,
			},
		},
		{
			name: `exported var`,
			obj: parseObject(t, `Toby`,
				`package jim
				var Toby float64`),
			want: Info{
				objectFilter: `jim.Toby`,
			},
		},
		{
			name: `exported const`,
			obj: parseObject(t, `Ludo`,
				`package jim
				const Ludo int = 42`),
			want: Info{
				objectFilter: `jim.Ludo`,
			},
		},
		{
			name: `label`,
			obj: parseObject(t, `Gobo`,
				`package jim
				func main() {
					i := 0
				Gobo:
					i++
					if i < 10 {
						goto Gobo
					}
				}`),
			want: Info{
				objectFilter: `jim.Gobo`,
			},
		},
		{
			name: `exported specific type`,
			obj: parseObject(t, `Jen`,
				`package jim
				type Jen struct{}`),
			want: Info{
				objectFilter: `jim.Jen`,
			},
		},
		{
			name: `exported generic type`,
			obj: parseObject(t, `Henson`,
				`package jim
				type Henson[T comparable] struct{}`),
			want: Info{
				objectFilter: `jim.Henson[comparable]`,
			},
		},
		{
			name: `exported specific function`,
			obj: parseObject(t, `Jareth`,
				`package jim
				func Jareth() {}`),
			want: Info{
				objectFilter: `jim.Jareth`,
			},
		},
		{
			name: `exported generic function`,
			obj: parseObject(t, `Didymus`,
				`package jim
				func Didymus[T comparable]() {}`),
			want: Info{
				objectFilter: `jim.Didymus[comparable]`,
			},
		},
		{
			name: `exported specific method`,
			obj: parseObject(t, `Kira`,
				`package jim
				type Fizzgig string
				func (f Fizzgig) Kira() {}`),
			want: Info{
				objectFilter: `jim.Fizzgig`,
			},
		},
		{
			name: `unexported specific method without parameters or results`,
			obj: parseObject(t, `frank`,
				`package jim
				type Aughra int
				func (a Aughra) frank() {}`),
			want: Info{
				objectFilter: `jim.Aughra`,
				methodFilter: `jim.frank()`,
			},
		},
		{
			name: `unexported specific method with parameters and results`,
			obj: parseObject(t, `frank`,
				`package jim
				type Aughra int
				func (a Aughra) frank(other Aughra) (bool, error) {
					return a == other, nil
				}`),
			want: Info{
				objectFilter: `jim.Aughra`,
				methodFilter: `jim.frank(jim.Aughra)(bool, error)`,
			},
		},
		{
			name: `unexported specific method with variadic parameter`,
			obj: parseObject(t, `frank`,
				`package jim
				type Aughra int
				func (a Aughra) frank(others ...Aughra) int {
					return len(others) + 1
				}`),
			want: Info{
				objectFilter: `jim.Aughra`,
				methodFilter: `jim.frank(...jim.Aughra) int`,
			},
		},
		{
			name: `unexported generic method with type parameters and instance argument`,
			obj: parseObject(t, `frank`,
				`package jim
				type Aughra[T ~float64] struct {
					value T
				}
				func (a *Aughra[T]) frank(other *Aughra[float64]) bool {
					return float64(a.value) == other.value
				}`),
			want: Info{
				objectFilter: `jim.Aughra[~float64]`,
				methodFilter: `jim.frank(*jim.Aughra[float64]) bool`,
			},
		},
		{
			name: `unexported generic method with type parameters and generic argument`,
			obj: parseObject(t, `frank`,
				`package jim
				type Aughra[T ~float64] struct {
					value T
				}
				func (a *Aughra[T]) frank(other *Aughra[T]) bool {
					return a.value == other.value
				}`),
			want: Info{
				objectFilter: `jim.Aughra[~float64]`,
				methodFilter: `jim.frank(*jim.Aughra[~float64]) bool`,
			},
		},
		{
			name: `specific method on unexported type`,
			obj: parseObject(t, `Red`,
				`package jim
				type wembley struct{}
				func (w wembley) Red() {}`),
			want: Info{
				objectFilter: `jim.wembley`,
			},
		},
		{
			name: `unexported method resulting in an interface with exported methods`,
			obj: parseObject(t, `bear`,
				`package jim
				type Fozzie struct{}
				func (f *Fozzie) bear() interface{
					WakkaWakka(joke string)(landed bool)
					Firth()(string, error)
				}`),
			want: Info{
				objectFilter: `jim.Fozzie`,
				methodFilter: `jim.bear() interface{ Firth()(string, error); WakkaWakka(string) bool }`,
			},
		},
		{
			name: `unexported method resulting in an interface with unexported methods`,
			obj: parseObject(t, `bear`,
				`package jim
				type Fozzie struct{}
				func (f *Fozzie) bear() interface{
					wakkaWakka(joke string)(landed bool)
					firth()(string, error)
				}`),
			want: Info{
				objectFilter: `jim.Fozzie`,
				// The package path, i.e. `jim.`, is used on unexported methods
				// to ensure the filter will not match another package's method.
				methodFilter: `jim.bear() interface{ jim.firth()(string, error); jim.wakkaWakka(string) bool }`,
			},
		},
		{
			name: `unexported method resulting in an empty interface `,
			obj: parseObject(t, `bear`,
				`package jim
				type Fozzie struct{}
				func (f *Fozzie) bear() interface{}`),
			want: Info{
				objectFilter: `jim.Fozzie`,
				methodFilter: `jim.bear() any`,
			},
		},
		{
			name: `unexported method resulting in a function`,
			obj: parseObject(t, `bear`,
				`package jim
				type Fozzie struct{}
				func (f *Fozzie) bear() func(joke string)(landed bool)`),
			want: Info{
				objectFilter: `jim.Fozzie`,
				methodFilter: `jim.bear() func(string) bool`,
			},
		},
		{
			name: `unexported method resulting in a struct`,
			obj: parseObject(t, `bear`,
				`package jim
				type Fozzie struct{}
				func (f *Fozzie) bear() struct{
					Joke string
					WakkaWakka bool
				}`),
			want: Info{
				objectFilter: `jim.Fozzie`,
				methodFilter: `jim.bear() struct{ Joke string; WakkaWakka bool }`,
			},
		},
		{
			name: `unexported method resulting in a struct with type parameter`,
			obj: parseObject(t, `bear`,
				`package jim
				type Fozzie[T ~string|~int] struct{}
				func (f *Fozzie[T]) bear() struct{
					Joke T
					wakkaWakka bool
				}`),
			want: Info{
				objectFilter: `jim.Fozzie[~int|~string]`,
				// The `Joke ~int|~string` part will likely not match other methods
				// such as methods with `Joke string` or `Joke int`, however the
				// interface should be defined for the instantiations of this type
				// and those should have the correct field type for `Joke`.
				methodFilter: `jim.bear() struct{ Joke ~int|~string; jim.wakkaWakka bool }`,
			},
		},
		{
			name: `unexported method resulting in an empty struct`,
			obj: parseObject(t, `bear`,
				`package jim
				type Fozzie struct{}
				func (f *Fozzie) bear() struct{}`),
			want: Info{
				objectFilter: `jim.Fozzie`,
				methodFilter: `jim.bear() struct{}`,
			},
		},
		{
			name: `unexported method resulting in a slice`,
			obj: parseObject(t, `bear`,
				`package jim
				type Fozzie struct{}
				func (f *Fozzie) bear()(jokes []string)`),
			want: Info{
				objectFilter: `jim.Fozzie`,
				methodFilter: `jim.bear() []string`,
			},
		},
		{
			name: `unexported method resulting in an array`,
			obj: parseObject(t, `bear`,
				`package jim
				type Fozzie struct{}
				func (f *Fozzie) bear()(jokes [2]string)`),
			want: Info{
				objectFilter: `jim.Fozzie`,
				methodFilter: `jim.bear() [2]string`,
			},
		},
		{
			name: `unexported method resulting in a map`,
			obj: parseObject(t, `bear`,
				`package jim
				type Fozzie struct{}
				func (f *Fozzie) bear()(jokes map[string]bool)`),
			want: Info{
				objectFilter: `jim.Fozzie`,
				methodFilter: `jim.bear() map[string]bool`,
			},
		},
		{
			name: `unexported method resulting in a channel`,
			obj: parseObject(t, `bear`,
				`package jim
				type Fozzie struct{}
				func (f *Fozzie) bear() chan string`),
			want: Info{
				objectFilter: `jim.Fozzie`,
				methodFilter: `jim.bear() chan string`,
			},
		},
		{
			name: `unexported method resulting in a complex compound named type`,
			obj: parseObject(t, `packRat`,
				`package jim
				type Gonzo[T any] struct{
					v T
				}
				func (g Gonzo[T]) Get() T { return g.v }
				type Rizzo struct{}
				func (r Rizzo) packRat(v int) Gonzo[Gonzo[Gonzo[int]]] {
					return Gonzo[Gonzo[Gonzo[int]]]{v: Gonzo[Gonzo[int]]{v: Gonzo[int]{v: v}}}
				}
				var _ int = Rizzo{}.packRat(42).Get().Get().Get()`),
			want: Info{
				objectFilter: `jim.Rizzo`,
				methodFilter: `jim.packRat(int) jim.Gonzo[jim.Gonzo[jim.Gonzo[int]]]`,
			},
		},
		{
			name: `unexported method resulting in an instance with same type parameter`,
			obj: parseObject(t, `sidekick`,
				`package jim
				type Beaker[T any] struct{}
				type Honeydew[S any] struct{}
				func (hd Honeydew[S]) sidekick() Beaker[S] {
					return Beaker[S]{}
				}`),
			want: Info{
				objectFilter: `jim.Honeydew[any]`,
				methodFilter: `jim.sidekick() jim.Beaker[any]`,
			},
		},
		{
			name: `struct with self referencing type parameter constraints`,
			obj: parseObject(t, `Keys`,
				`package jim
				func Keys[K comparable, V any, M ~map[K]V](m M) []K {
					keys := make([]K, 0, len(m))
					for k := range m {
						keys = append(keys, k)
					}
					return keys
				}`),
			want: Info{
				objectFilter: `jim.Keys[comparable, any, ~map[comparable]any]`,
			},
		},
		{
			name: `interface with self referencing type parameter constraints`,
			obj: parseObject(t, `ElectricMayhem`,
				`package jim
				type ElectricMayhem[K comparable, V any, M ~map[K]V] interface {
					keys() []K
					values() []V
					asMap() M
				}`),
			want: Info{
				objectFilter: `jim.ElectricMayhem[comparable, any, ~map[comparable]any]`,
			},
		},
		{
			name: `function with recursive referencing type parameter constraints`,
			obj: parseObject(t, `doWork`,
				`package jim
				type Doozer[T any] interface {
					comparable
					Work() T
				}

				func doWork[T Doozer[T]](a T) T {
					return a.Work()
				}`),
			want: Info{
				objectFilter: `jim.doWork[jim.Doozer[jim.Doozer[...]]]`,
			},
		},
		{
			name: `function with recursive referencing multiple type parameter constraints`,
			obj: parseObject(t, `doWork`,
				`package jim
				type Doozer[T, U any] interface {
					Work() T
					Play() U
				}

				func doWork[T Doozer[T, U], U any](a T) T {
					return a.Work()
				}`),
			want: Info{
				objectFilter: `jim.doWork[jim.Doozer[jim.Doozer[...], any], any]`,
			},
		},
		{
			name: `function with multiple recursive referencing multiple type parameter constraints`,
			obj: parseObject(t, `doWork`,
				`package jim
				type Doozer[T, U any] interface {
					Work() T
					Play() U
				}

				func doWork[T Doozer[T, U], U Doozer[T, U]](a T) T {
					return a.Work()
				}`),
			want: Info{
				objectFilter: `jim.doWork[jim.Doozer[jim.Doozer[...], jim.Doozer[...]], jim.Doozer[jim.Doozer[...], jim.Doozer[...]]]`,
			},
		},
		{
			name: `function with multiple recursive referencing type parameter constraints`,
			obj: parseObject(t, `doWork`,
				`package jim
				type Doozer[T any] interface {
					Work() T
				}

				type Fraggle[U any] interface {
					Play() U
				}

				func doWork[T Doozer[T], U Fraggle[U]](a T) T {
					return a.Work()
				}`),
			want: Info{
				objectFilter: `jim.doWork[jim.Doozer[jim.Doozer[...]], jim.Fraggle[jim.Fraggle[...]]]`,
			},
		},
		{
			name: `function with osculating recursive referencing type parameter constraints`,
			obj: parseObject(t, `doWork`,
				`package jim
				type Doozer[T any] interface {
					Work() T
				}

				type Fraggle[U any] interface {
					Play() U
				}

				func doWork[T Doozer[U], U Fraggle[T]]() {}`),
			want: Info{
				objectFilter: `jim.doWork[jim.Doozer[jim.Fraggle[jim.Doozer[...]]], jim.Fraggle[jim.Doozer[jim.Fraggle[...]]]]`,
			},
		},
	}

	t.Run(`SetName`, func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				d := &testDecl{}
				equal(t, d.Dce().unnamed(), true)
				equal(t, d.Dce().String(), `[unnamed] -> []`)
				t.Log(`object:`, types.ObjectString(tt.obj, nil))

				d.Dce().SetName(tt.obj)
				equal(t, d.Dce().unnamed(), tt.want.unnamed())
				equal(t, d.Dce().objectFilter, tt.want.objectFilter)
				equal(t, d.Dce().methodFilter, tt.want.methodFilter)
				equalSlices(t, d.Dce().getDeps(), tt.want.getDeps())
				equal(t, d.Dce().String(), tt.want.String())
			})
		}
	})

	t.Run(`addDep`, func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				d := &testDecl{}
				t.Log(`object:`, types.ObjectString(tt.obj, nil))

				wantDeps := []string{}
				if len(tt.want.objectFilter) > 0 {
					wantDeps = append(wantDeps, tt.want.objectFilter)
				}
				if len(tt.want.methodFilter) > 0 {
					wantDeps = append(wantDeps, tt.want.methodFilter)
				}
				sort.Strings(wantDeps)

				c := Collector{}
				c.CollectDCEDeps(d, func() {
					c.DeclareDCEDep(tt.obj)
				})
				equalSlices(t, d.Dce().getDeps(), wantDeps)
			})
		}
	})
}

func Test_Info_SetNameOnlyOnce(t *testing.T) {
	pkg := testPackage(`mogwai`)
	obj1 := quickVar(pkg, `Gizmo`)
	obj2 := quickVar(pkg, `Stripe`)

	decl := &testDecl{}
	decl.Dce().SetName(obj1)

	err := capturePanic(t, func() {
		decl.Dce().SetName(obj2)
	})
	errorMatches(t, err, `^may only set the name once for path/to/mogwai\.Gizmo .*$`)
}

func Test_Info_UsesDeps(t *testing.T) {
	tests := []struct {
		name     string
		id       string // identifier to check for usage and instance
		line     int    // line number to find the identifier on
		src      string
		wantDeps []string
	}{
		{
			name: `usage of specific struct`,
			id:   `Sinclair`,
			line: 5,
			src: `package epsilon3
				type Sinclair struct{}
				func (s Sinclair) command() { }
				func main() {
					Sinclair{}.command() //<-- line 5
				}`,
			wantDeps: []string{`epsilon3.Sinclair`},
		},
		{
			name: `usage of generic struct`,
			id:   `Sheridan`,
			line: 5,
			src: `package epsilon3
				type Sheridan[T comparable] struct{}
				func (s Sheridan[T]) command() { }
				func main() {
					Sheridan[string]{}.command() //<-- line 5
				}`,
			wantDeps: []string{`epsilon3.Sheridan[string]`},
		},
		{
			name: `usage of unexported method of generic struct`,
			id:   `command`,
			line: 5,
			src: `package epsilon3
				type Sheridan[T comparable] struct{}
				func (s Sheridan[T]) command() { }
				func main() {
					Sheridan[string]{}.command() //<-- line 5
				}`,
			// unexported methods need the method filter for matching with
			// unexported methods on interfaces.
			wantDeps: []string{
				`epsilon3.Sheridan[string]`,
				`epsilon3.command()`,
			},
		},
		{
			name: `usage of unexported method of generic struct pointer`,
			id:   `command`,
			line: 5,
			src: `package epsilon3
				type Sheridan[T comparable] struct{}
				func (s *Sheridan[T]) command() { }
				func main() {
					(&Sheridan[string]{}).command() //<-- line 5
				}`,
			// unexported methods need the method filter for matching with
			// unexported methods on interfaces.
			wantDeps: []string{
				`epsilon3.Sheridan[string]`,
				`epsilon3.command()`,
			},
		},
		{
			name: `invocation of function with implicit type arguments`,
			id:   `Move`,
			line: 5,
			src: `package epsilon3
				type Ivanova[T any] struct{}
				func Move[T ~string|~int](i Ivanova[T]) { }
				func main() {
					Move(Ivanova[string]{}) //<-- line 5
				}`,
			wantDeps: []string{`epsilon3.Move[string]`},
		},
		{
			name: `exported method on a complex generic type`,
			id:   `Get`,
			line: 6,
			src: `package epsilon3
				type Garibaldi[T any] struct{ v T }
				func (g Garibaldi[T]) Get() T { return g.v }
				func main() {
					michael := Garibaldi[Garibaldi[Garibaldi[int]]]{v: Garibaldi[Garibaldi[int]]{v: Garibaldi[int]{v: 42}}}
					_ = michael.Get() // <-- line 6
				}`,
			wantDeps: []string{`epsilon3.Garibaldi[epsilon3.Garibaldi[epsilon3.Garibaldi[int]]]`},
		},
		{
			name: `unexported method on a complex generic type`,
			id:   `get`,
			line: 6,
			src: `package epsilon3
				type Garibaldi[T any] struct{ v T }
				func (g Garibaldi[T]) get() T { return g.v }
				func main() {
					michael := Garibaldi[Garibaldi[Garibaldi[int]]]{v: Garibaldi[Garibaldi[int]]{v: Garibaldi[int]{v: 42}}}
					_ = michael.get() // <-- line 6
				}`,
			wantDeps: []string{
				`epsilon3.Garibaldi[epsilon3.Garibaldi[epsilon3.Garibaldi[int]]]`,
				`epsilon3.get() epsilon3.Garibaldi[epsilon3.Garibaldi[int]]`,
			},
		},
		{
			name: `invoke of method with an unnamed interface receiver`,
			id:   `heal`,
			line: 8,
			src: `package epsilon3
				type Franklin struct{}
				func (g Franklin) heal() {}
				func main() {
					var stephen interface{
						heal()
					} = Franklin{}
					stephen.heal() // <-- line 8
				}`,
			wantDeps: []string{
				`epsilon3.heal()`,
			},
		},
		{
			name: `invoke a method with a generic return type via instance`,
			// Based on go/1.19.13/x64/test/dictionaryCapture-noinline.go
			id:   `lennier`,
			line: 6,
			src: `package epsilon3								
				type delenn[T any] struct { a T }
				func (d delenn[T]) lennier() T { return d.a }
				func cocoon() int {
					x := delenn[int]{a: 7}
					f := delenn[int].lennier // <-- line 6
					return f(x)
				}`,
			wantDeps: []string{
				`epsilon3.delenn[int]`,
				`epsilon3.lennier() int`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &testDecl{}
			uses, inst := parseInstanceUse(t, tt.line, tt.id, tt.src)
			tArgs := typeListToSlice(inst.TypeArgs)
			t.Logf(`object: %s with [%s]`, types.ObjectString(uses, nil), (typesutil.TypeList)(tArgs).String())

			c := Collector{}
			c.CollectDCEDeps(d, func() {
				c.DeclareDCEDep(uses, tArgs...)
			})
			equalSlices(t, d.Dce().getDeps(), tt.wantDeps)
		})
	}
}

func Test_Info_SpecificCasesDeps(t *testing.T) {
	tests := []struct {
		name     string
		obj      types.Object
		tArgs    []types.Type
		wantDeps []string
	}{
		{
			name: `struct instantiation with generic object`,
			obj: parseObject(t, `Mikey`,
				`package astoria;
				type Mikey[T comparable] struct{}
				`),
			tArgs:    []types.Type{types.Typ[types.String]},
			wantDeps: []string{`astoria.Mikey[string]`},
		},
		{
			name: `method instantiation with generic object`,
			obj: parseObject(t, `brand`,
				`package astoria;
				type Mikey[T comparable] struct{ a T}
				func (m Mikey[T]) brand() T {
					return m.a
				}`),
			tArgs: []types.Type{types.Typ[types.String]},
			wantDeps: []string{
				`astoria.Mikey[string]`,
				`astoria.brand() string`,
			},
		},
		{
			name: `method instantiation with generic object and multiple type parameters`,
			obj: parseObject(t, `shuffle`,
				`package astoria;
				type Chunk[K comparable, V any] struct{ data map[K]V }
				func (c Chunk[K, V]) shuffle(k K) V {
					return c.data[k]
				}`),
			tArgs: []types.Type{types.Typ[types.String], types.Typ[types.Int]},
			wantDeps: []string{
				`astoria.Chunk[string, int]`,
				`astoria.shuffle(string) int`,
			},
		},
		{
			name: `method instantiation with generic object renamed type parameters`,
			obj: parseObject(t, `shuffle`,
				`package astoria;
				type Chunk[K comparable, V any] struct{ data map[K]V }
				func (c Chunk[T, K]) shuffle(k T) K {
					return c.data[k]
				}`),
			tArgs: []types.Type{types.Typ[types.String], types.Typ[types.Int]},
			wantDeps: []string{
				`astoria.Chunk[string, int]`,
				`astoria.shuffle(string) int`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &testDecl{}
			t.Logf(`object: %s with [%s]`, types.ObjectString(tt.obj, nil), (typesutil.TypeList)(tt.tArgs).String())

			c := Collector{}
			c.CollectDCEDeps(d, func() {
				c.DeclareDCEDep(tt.obj, tt.tArgs...)
			})
			equalSlices(t, d.Dce().getDeps(), tt.wantDeps)
		})
	}
}

func Test_Info_SetAsAlive(t *testing.T) {
	pkg := testPackage(`fantasia`)

	t.Run(`set alive prior to naming`, func(t *testing.T) {
		obj := quickVar(pkg, `Falkor`)
		decl := &testDecl{}
		equal(t, decl.Dce().isAlive(), true) // unnamed is automatically alive
		equal(t, decl.Dce().String(), `[unnamed] -> []`)

		decl.Dce().SetAsAlive()
		equal(t, decl.Dce().isAlive(), true) // still alive but now explicitly alive
		equal(t, decl.Dce().String(), `[alive] [unnamed] -> []`)

		decl.Dce().SetName(obj)
		equal(t, decl.Dce().isAlive(), true) // alive because SetAsAlive was called
		equal(t, decl.Dce().String(), `[alive] path/to/fantasia.Falkor -> []`)
	})

	t.Run(`set alive after naming`, func(t *testing.T) {
		obj := quickVar(pkg, `Artax`)
		decl := &testDecl{}
		equal(t, decl.Dce().isAlive(), true) // unnamed is automatically alive
		equal(t, decl.Dce().String(), `[unnamed] -> []`)

		decl.Dce().SetName(obj)
		equal(t, decl.Dce().isAlive(), false) // named so no longer automatically alive
		equal(t, decl.Dce().String(), `path/to/fantasia.Artax -> []`)

		decl.Dce().SetAsAlive()
		equal(t, decl.Dce().isAlive(), true) // alive because SetAsAlive was called
		equal(t, decl.Dce().String(), `[alive] path/to/fantasia.Artax -> []`)
	})
}

func Test_Selector_JustVars(t *testing.T) {
	pkg := testPackage(`tolkien`)
	frodo := quickTestDecl(quickVar(pkg, `Frodo`))
	samwise := quickTestDecl(quickVar(pkg, `Samwise`))
	meri := quickTestDecl(quickVar(pkg, `Meri`))
	pippin := quickTestDecl(quickVar(pkg, `Pippin`))
	aragorn := quickTestDecl(quickVar(pkg, `Aragorn`))
	boromir := quickTestDecl(quickVar(pkg, `Boromir`))
	gimli := quickTestDecl(quickVar(pkg, `Gimli`))
	legolas := quickTestDecl(quickVar(pkg, `Legolas`))
	gandalf := quickTestDecl(quickVar(pkg, `Gandalf`))
	fellowship := []*testDecl{
		frodo, samwise, meri, pippin, aragorn,
		boromir, gimli, legolas, gandalf,
	}

	c := Collector{}
	c.CollectDCEDeps(frodo, func() {
		c.DeclareDCEDep(samwise.obj)
		c.DeclareDCEDep(meri.obj)
		c.DeclareDCEDep(pippin.obj)
	})
	c.CollectDCEDeps(pippin, func() {
		c.DeclareDCEDep(meri.obj)
	})
	c.CollectDCEDeps(aragorn, func() {
		c.DeclareDCEDep(boromir.obj)
	})
	c.CollectDCEDeps(gimli, func() {
		c.DeclareDCEDep(legolas.obj)
	})
	c.CollectDCEDeps(legolas, func() {
		c.DeclareDCEDep(gimli.obj)
	})
	c.CollectDCEDeps(gandalf, func() {
		c.DeclareDCEDep(frodo.obj)
		c.DeclareDCEDep(aragorn.obj)
		c.DeclareDCEDep(gimli.obj)
		c.DeclareDCEDep(legolas.obj)
	})

	for _, decl := range fellowship {
		equal(t, decl.Dce().isAlive(), false)
	}

	tests := []struct {
		name string
		init []*testDecl // which decls to set explicitly alive
		want []*testDecl // which decls should be determined as alive
	}{
		{
			name: `all alive`,
			init: fellowship,
			want: fellowship,
		},
		{
			name: `all dead`,
			init: []*testDecl{},
			want: []*testDecl{},
		},
		{
			name: `Frodo`,
			init: []*testDecl{frodo},
			want: []*testDecl{frodo, samwise, meri, pippin},
		},
		{
			name: `Sam and Pippin`,
			init: []*testDecl{samwise, pippin},
			want: []*testDecl{samwise, meri, pippin},
		},
		{
			name: `Gandalf`,
			init: []*testDecl{gandalf},
			want: fellowship,
		},
		{
			name: `Legolas`,
			init: []*testDecl{legolas},
			want: []*testDecl{legolas, gimli},
		},
		{
			name: `Gimli`,
			init: []*testDecl{gimli},
			want: []*testDecl{legolas, gimli},
		},
		{
			name: `Boromir`,
			init: []*testDecl{boromir},
			want: []*testDecl{boromir},
		},
		{
			name: `Aragorn`,
			init: []*testDecl{aragorn},
			want: []*testDecl{aragorn, boromir},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, decl := range fellowship {
				decl.Dce().alive = false
			}
			for _, decl := range tt.init {
				decl.Dce().SetAsAlive()
			}

			s := &Selector[*testDecl]{}
			for _, decl := range fellowship {
				s.Include(decl, false)
			}

			selected := s.AliveDecls()
			for _, decl := range tt.want {
				if _, ok := selected[decl]; !ok {
					t.Errorf(`expected %q to be alive`, decl.obj.String())
				}
				delete(selected, decl)
			}
			for decl := range selected {
				t.Errorf(`expected %q to be dead`, decl.obj.String())
			}
		})
	}
}

func Test_Selector_SpecificMethods(t *testing.T) {
	objects := parseObjects(t,
		`package pratchett

		type rincewind struct{}
		func (r rincewind) Run() {}
		func (r rincewind) hide() {}

		type Vimes struct{}
		func (v Vimes) Run() {}
		func (v Vimes) Read() {}

		func Vetinari() {}`)

	var (
		// Objects are in read order so pick the objects we want for this test
		// while skipping over `r rincewind` and `v Vimes`.
		rincewind     = quickTestDecl(objects[0])
		rincewindRun  = quickTestDecl(objects[2])
		rincewindHide = quickTestDecl(objects[4])
		vimes         = quickTestDecl(objects[5])
		vimesRun      = quickTestDecl(objects[7])
		vimesRead     = quickTestDecl(objects[9])
		vetinari      = quickTestDecl(objects[10])
	)
	allDecls := []*testDecl{rincewind, rincewindRun, rincewindHide, vimes, vimesRun, vimesRead, vetinari}

	c := Collector{}
	c.CollectDCEDeps(rincewindRun, func() {
		c.DeclareDCEDep(rincewind.obj)
	})
	c.CollectDCEDeps(rincewindHide, func() {
		c.DeclareDCEDep(rincewind.obj)
	})
	c.CollectDCEDeps(vimesRun, func() {
		c.DeclareDCEDep(vimes.obj)
	})
	c.CollectDCEDeps(vimesRead, func() {
		c.DeclareDCEDep(vimes.obj)
	})
	vetinari.Dce().SetAsAlive()

	tests := []struct {
		name string
		deps []*testDecl // which decls are vetinari dependent on
		want []*testDecl // which decls should be determined as alive
	}{
		{
			name: `no deps`,
			deps: []*testDecl{},
			want: []*testDecl{vetinari},
		},
		{
			name: `structs`,
			deps: []*testDecl{rincewind, vimes},
			// rincewindHide is not included because it is not exported and not used.
			want: []*testDecl{rincewind, rincewindRun, vimes, vimesRun, vimesRead, vetinari},
		},
		{
			name: `exported method`,
			deps: []*testDecl{rincewind, rincewindRun},
			want: []*testDecl{rincewind, rincewindRun, vetinari},
		},
		{
			name: `unexported method`,
			deps: []*testDecl{rincewind, rincewindHide},
			want: []*testDecl{rincewind, rincewindRun, rincewindHide, vetinari},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vetinari.Dce().deps = nil // reset deps
			c.CollectDCEDeps(vetinari, func() {
				for _, decl := range tt.deps {
					c.DeclareDCEDep(decl.obj)
				}
			})

			s := Selector[*testDecl]{}
			for _, decl := range allDecls {
				s.Include(decl, false)
			}
			selected := s.AliveDecls()
			for _, decl := range tt.want {
				if _, ok := selected[decl]; !ok {
					t.Errorf(`expected %q to be alive`, decl.obj.String())
				}
				delete(selected, decl)
			}
			for decl := range selected {
				t.Errorf(`expected %q to be dead`, decl.obj.String())
			}
		})
	}
}

type testDecl struct {
	obj types.Object // should match the object used in Dce.SetName when set
	dce Info
}

func (d *testDecl) Dce() *Info {
	return &d.dce
}

func testPackage(name string) *types.Package {
	return types.NewPackage(`path/to/`+name, name)
}

func quickTestDecl(o types.Object) *testDecl {
	d := &testDecl{obj: o}
	d.Dce().SetName(o)
	return d
}

func quickVar(pkg *types.Package, name string) *types.Var {
	return types.NewVar(token.NoPos, pkg, name, types.Typ[types.Int])
}

func newTypeInfo() *types.Info {
	return &types.Info{
		Defs:      map[*ast.Ident]types.Object{},
		Uses:      map[*ast.Ident]types.Object{},
		Instances: map[*ast.Ident]types.Instance{},
	}
}

func parseObject(t *testing.T, name, source string) types.Object {
	t.Helper()
	objects := parseObjects(t, source)
	for _, obj := range objects {
		if obj.Name() == name {
			return obj
		}
	}
	t.Fatalf(`object %q not found`, name)
	return nil
}

func parseObjects(t *testing.T, source string) []types.Object {
	t.Helper()
	fset := token.NewFileSet()
	info := newTypeInfo()
	parsePackage(t, source, fset, info)
	objects := make([]types.Object, 0, len(info.Defs))
	for _, obj := range info.Defs {
		if obj != nil {
			objects = append(objects, obj)
		}
	}
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].Pos() < objects[j].Pos()
	})
	return objects
}

func parseInstanceUse(t *testing.T, lineNo int, idName, source string) (types.Object, types.Instance) {
	t.Helper()
	fset := token.NewFileSet()
	info := newTypeInfo()
	parsePackage(t, source, fset, info)
	for id, obj := range info.Uses {
		if id.Name == idName && fset.Position(id.Pos()).Line == lineNo {
			return obj, info.Instances[id]
		}
	}
	t.Fatalf(`failed to find %s on line %d`, idName, lineNo)
	return nil, types.Instance{}
}

func parsePackage(t *testing.T, source string, fset *token.FileSet, info *types.Info) *types.Package {
	t.Helper()
	f, err := parser.ParseFile(fset, `test.go`, source, 0)
	if err != nil {
		t.Fatal(`parsing source:`, err)
	}

	conf := types.Config{
		Importer:                 importer.Default(),
		DisableUnusedImportCheck: true,
	}
	pkg, err := conf.Check(f.Name.Name, fset, []*ast.File{f}, info)
	if err != nil {
		t.Fatal(`type checking:`, err)
	}
	return pkg
}

func capturePanic(t *testing.T, f func()) (err error) {
	t.Helper()
	defer func() {
		t.Helper()
		if r := recover(); r != nil {
			if err2, ok := r.(error); ok {
				err = err2
				return
			}
			t.Errorf(`expected an error to be panicked but got (%[1]T) %[1]#v`, r)
			return
		}
		t.Error(`expected a panic but got none`)
	}()

	f()
	return nil
}

func errorMatches(t *testing.T, err error, wantPattern string) {
	t.Helper()
	re := regexp.MustCompile(wantPattern)
	if got := fmt.Sprint(err); !re.MatchString(got) {
		t.Errorf(`expected error %q to match %q`, got, re.String())
	}
}

func depCount(t *testing.T, decl *testDecl, want int) {
	t.Helper()
	if got := len(decl.Dce().deps); got != want {
		t.Errorf(`expected %d deps but got %d`, want, got)
	}
}

func equal[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("Unexpected value was gotten:\n\texp: %#v\n\tgot: %#v", want, got)
	}
}

func equalSlices[T comparable](t *testing.T, got, want []T) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("expected %d but got %d\n\texp: %#v\n\tgot: %#v", len(want), len(got), want, got)
		return
	}
	for i, wantElem := range want {
		equal(t, got[i], wantElem)
	}
}
