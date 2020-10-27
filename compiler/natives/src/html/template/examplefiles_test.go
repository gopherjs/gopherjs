// +build js

package template_test

import (
	"fmt"
)

func ExampleTemplate_glob() {
	fmt.Print("T0 invokes T1: (T1 invokes T2: (This is T2))")
	// Output:
	// T0 invokes T1: (T1 invokes T2: (This is T2))
}

func ExampleTemplate_helpers() {
	fmt.Print("Driver 1 calls T1: (T1 invokes T2: (This is T2))\nDriver 2 calls T2: (This is T2)")
	// Output:
	// Driver 1 calls T1: (T1 invokes T2: (This is T2))
	// Driver 2 calls T2: (This is T2)
}

func ExampleTemplate_share() {
	fmt.Print("T0 (second version) invokes T1: (T1 invokes T2: (T2, version B))\nT0 (first version) invokes T1: (T1 invokes T2: (T2, version A))")
	// Output:
	// T0 (second version) invokes T1: (T1 invokes T2: (T2, version B))
	// T0 (first version) invokes T1: (T1 invokes T2: (T2, version A))
}
