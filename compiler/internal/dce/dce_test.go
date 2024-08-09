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

	// The second collection overwrites the first collection.
	c.CollectDCEDeps(decl2, func() {
		c.DeclareDCEDep(obj5)
	})
	depCount(t, decl1, 2)
	depCount(t, decl2, 1)
}

func Test_Info_SetNameAndDep(t *testing.T) {
	tests := []struct {
		name          string
		obj           types.Object
		wantObjFilter string // expected Info after SetName
		wantMetFilter string // expected Info after SetName
		wantDep       string // expected dep after addDep
	}{
		{
			name: `package`,
			obj: parseObject(t, `Sarah`,
				`package jim
				import Sarah "fmt"`),
			wantObjFilter: `jim.Sarah`,
			wantDep:       `jim.Sarah`,
		},
		{
			name: `exposed var`,
			obj: parseObject(t, `Toby`,
				`package jim
				var Toby float64`),
			wantObjFilter: `jim.Toby`,
			wantDep:       `jim.Toby`,
		},
		{
			name: `exposed const`,
			obj: parseObject(t, `Ludo`,
				`package jim
				const Ludo int = 42`),
			wantObjFilter: `jim.Ludo`,
			wantDep:       `jim.Ludo`,
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
			wantObjFilter: `jim.Gobo`,
			wantDep:       `jim.Gobo`,
		},
		{
			name: `exposed specific type`,
			obj: parseObject(t, `Jen`,
				`package jim
				type Jen struct{}`),
			wantObjFilter: `jim.Jen`,
			wantDep:       `jim.Jen`,
		},
		{
			name: `exposed generic type`,
			obj: parseObject(t, `Henson`,
				`package jim
				type Henson[T comparable] struct{}`),
			wantObjFilter: `jim.Henson`,
			wantDep:       `jim.Henson`,
		},
		{
			name: `exposed specific function`,
			obj: parseObject(t, `Jareth`,
				`package jim
				func Jareth() {}`),
			wantObjFilter: `jim.Jareth`,
			wantDep:       `jim.Jareth`,
		},
		{
			name: `exposed generic function`,
			obj: parseObject(t, `Didymus`,
				`package jim
				func Didymus[T comparable]() {}`),
			wantObjFilter: `jim.Didymus`,
			wantDep:       `jim.Didymus`,
		},
		{
			name: `exposed specific method`,
			obj: parseObject(t, `Kira`,
				`package jim
				type Fizzgig string
				func (f Fizzgig) Kira() {}`),
			wantObjFilter: `jim.Fizzgig`,
			wantDep:       `jim.Kira~`,
		},
		{
			name: `unexposed specific method`,
			obj: parseObject(t, `frank`,
				`package jim
				type Aughra int
				func (a Aughra) frank() {}`),
			wantObjFilter: `jim.Aughra`,
			wantMetFilter: `jim.frank~`,
			wantDep:       `jim.frank~`,
		},
		{
			name: `specific method on unexposed type`,
			obj: parseObject(t, `Red`,
				`package jim
				type wembley struct{}
				func (w wembley) Red() {}`),
			wantObjFilter: `jim.wembley`,
			wantDep:       `jim.Red~`,
		},
	}

	t.Run(`SetName`, func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				d := &testDecl{}
				equal(t, d.Dce().unnamed(), true)
				equal(t, d.Dce().String(), `[unnamed] . -> []`)
				t.Log(`object:`, types.ObjectString(tt.obj, nil))

				d.Dce().SetName(tt.obj)
				equal(t, d.Dce().unnamed(), false)
				objectFilter, methodFilter := d.Dce().getInfoNames()
				equal(t, objectFilter, tt.wantObjFilter)
				equal(t, methodFilter, tt.wantMetFilter)
			})
		}
	})

	t.Run(`addDep`, func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				d1 := &testDecl{}
				t.Log(`object:`, types.ObjectString(tt.obj, nil))

				d1.Dce().addDep(tt.obj)
				equal(t, len(d1.Dce().deps), 1)
				depNames := d1.Dce().getDepNames()
				equal(t, len(depNames), 1)
				equal(t, depNames[0], tt.wantDep)
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

func Test_Info_SetAsAlive(t *testing.T) {
	pkg := testPackage(`fantasia`)

	t.Run(`set alive prior to naming`, func(t *testing.T) {
		obj := quickVar(pkg, `Falkor`)
		decl := &testDecl{}
		equal(t, decl.Dce().isAlive(), true) // unnamed is automatically alive
		equal(t, decl.Dce().String(), `[unnamed] . -> []`)

		decl.Dce().SetAsAlive()
		equal(t, decl.Dce().isAlive(), true) // still alive but now explicitly alive
		equal(t, decl.Dce().String(), `[alive] [unnamed] . -> []`)

		decl.Dce().SetName(obj)
		equal(t, decl.Dce().isAlive(), true) // alive because SetAsAlive was called
		equal(t, decl.Dce().String(), `[alive] path/to/fantasia.Falkor -> []`)
	})

	t.Run(`set alive after naming`, func(t *testing.T) {
		obj := quickVar(pkg, `Artax`)
		decl := &testDecl{}
		equal(t, decl.Dce().isAlive(), true) // unnamed is automatically alive
		equal(t, decl.Dce().String(), `[unnamed] . -> []`)

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
			name: `exposed method`,
			deps: []*testDecl{rincewind, rincewindRun},
			want: []*testDecl{rincewind, rincewindRun, vetinari},
		},
		{
			name: `unexposed method`,
			deps: []*testDecl{rincewind, rincewindHide},
			want: []*testDecl{rincewind, rincewindRun, rincewindHide, vetinari},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	info := &types.Info{
		Defs: map[*ast.Ident]types.Object{},
	}
	parseInfo(t, source, info)
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

func parseInfo(t *testing.T, source string, info *types.Info) *types.Package {
	t.Helper()
	fset := token.NewFileSet()
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
		t.Errorf(`expected %#v but got %#v`, want, got)
	}
}
