// This tests an issue where a pointer to a structure is cast into
// and out of a proxy causing the resulting pointer to not contain the
// methods that were on the pointer structure originally.
package main

type Cat struct{ name string }

func (c *Cat) getName() string { return c.name }

type CatPtr *Cat

func sayHello(c *Cat) { println(`hello ` + c.getName()) }

func main() {
	a := &Cat{name: `mittens`}

	b := CatPtr(a) // `b` can not call `getName()` because "b.getName undefined (type CatPtr has no field or method getName)"
	sayHello(b)    // implicit cast of `b` to `*Cat` so it can call `getName()`.

	c := (*Cat)(b) // explicit cast of `b` to `*Cat` so it can also call `getName()`.
	println(c.getName() + ` says meow`)
}
