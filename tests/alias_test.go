package tests

import (
	"fmt"
)

type foo struct {
	a int
}

func bar() *foo {
	fmt.Println("bar() called")
	return &foo{ 42 }
}

func Example1() {
	q := &bar().a
	fmt.Println("pointer created")
	*q = 40
	fmt.Println(*q)
	// Output:
	// bar() called
	// pointer created
	// 40
}

func Example2() {
	f := foo{}
	p := &f.a
	f = foo{}
	f.a = 4
	fmt.Println(*p)
	// Output: 4
}

func Example3() {
	f := foo{}
	p := &f
	f = foo{ 4 }
	fmt.Println(p.a)
	// Output: 4
}

func Example4() {
	f := struct{
		a struct{
			b int
		}
	}{}
	p := &f.a
	q := &p.b
	r := &(*p).b
	*r = 4
	p = nil
	fmt.Println(*r)
	fmt.Println(*q)
	// Output:
	// 4
	// 4
}

func Example5() {
	f := struct{
		a [3]int
	}{ [3]int{6, 6, 6} }
	s := f.a[:]
	f.a = [3]int{4, 4, 4}
	fmt.Println(s[1])
	// Output:
	// 4
}
