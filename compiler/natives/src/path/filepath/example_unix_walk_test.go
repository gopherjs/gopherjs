// +build js
// +build !windows,!plan9

package filepath_test

import (
	"fmt"
)

func ExampleWalk() {
	fmt.Print(`On Unix:
visited file or dir: "."
visited file or dir: "dir"
visited file or dir: "dir/to"
visited file or dir: "dir/to/walk"
skipping a dir without errors: skip`)
	// Output:
	// On Unix:
	// visited file or dir: "."
	// visited file or dir: "dir"
	// visited file or dir: "dir/to"
	// visited file or dir: "dir/to/walk"
	// skipping a dir without errors: skip
}
