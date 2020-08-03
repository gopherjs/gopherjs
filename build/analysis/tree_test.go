package analysis

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestTreeNodeOrganize(t *testing.T) {
	io := &treeMapNode{Label: "io", Size: 10 * 1024}
	strconv := &treeMapNode{Label: "strconv", Size: 1 * 1024}
	fmt := &treeMapNode{Label: "fmt", Size: 50 * 1024, Children: nodeGroup{io, strconv}}
	main := &treeMapNode{Label: "main", Size: 1024 * 1024, Children: nodeGroup{io, fmt}}

	main.organizeTree()

	// Text representation of the spanning tree for the acyclic dependency graph.
	// Note that "io" is assigned as effective child of "main", since this is a
	// shorter path to the tree root.
	want := "" +
		"  main — 1.0 MiB own size, 1.1 MiB with children\n" +
		"  * fmt — 50 KiB own size, 51 KiB with children\n" +
		"  * * strconv — 1.0 KiB own size, 1.0 KiB with children\n" +
		"  * io — 10 KiB own size, 10 KiB with children\n"
	got := main.String()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("main.organizeTree() produced unexpected structure (-want,+got):\n%s", diff)
	}
}

func TestTreeNodeToSVG(t *testing.T) {
	// Build a dependency graph for a very simple project.
	cpu := &treeMapNode{Label: "internal/cpu", Size: 342}
	bytealg := &treeMapNode{Label: "internal/bytealg", Size: 468, Children: nodeGroup{cpu}}
	sys := &treeMapNode{Label: "runtime/internal/sys", Size: 350}
	js := &treeMapNode{Label: "gopherjs/js", Size: 5.2 * 1024}
	runtime := &treeMapNode{Label: "runtime", Size: 3.8 * 1024, Children: nodeGroup{js, bytealg, sys}}
	prelude := &treeMapNode{Label: "$prelude", Size: 32 * 1024}
	main := &treeMapNode{Label: "main", Size: 434}
	artifact := &treeMapNode{Label: "$artifact", Size: 0, Children: nodeGroup{prelude, runtime, main}}

	artifact.organizeTree()
	b := &bytes.Buffer{}
	if err := artifact.toSVG(b, newRect(1000, 1000)); err != nil {
		t.Fatalf("artifact.toSVG() returned error: %s", err)
	}

	want := `<g><rect x="0" y="0" width="1000" height="1000" fill="#fd7bff" fill-opacity="0.1" stroke="black" stroke-width="0.5"><title>$artifact — 0 B own size, 43 KiB with children</title></rect></g>
<g></g>
<g><rect x="10" y="10" width="736.9" height="980" fill="#a898ff" fill-opacity="0.1" stroke="black" stroke-width="0.5"><title>$prelude — 32 KiB own size, 32 KiB with children</title></rect></g>
<g><text x="378.45" y="500" font-size="20" transform="rotate(270 378.45 500.00)" text-anchor="middle" dominant-baseline="middle">$prelude — 32 KiB</text></g>
<g><rect x="746.9" y="10" width="233.34" height="980" fill="#e886ff" fill-opacity="0.1" stroke="black" stroke-width="0.5"><title>runtime — 3.8 KiB own size, 10 KiB with children</title></rect></g>
<g><text x="794.3" y="500" font-size="20" transform="rotate(270 794.30 500.00)" text-anchor="middle" dominant-baseline="middle">runtime — 3.8 KiB</text></g>
<g><rect x="835.87" y="15.83" width="138.54" height="795.12" fill="#0cf3b7" fill-opacity="0.1" stroke="black" stroke-width="0.5"><title>gopherjs/js — 5.2 KiB own size, 5.2 KiB with children</title></rect></g>
<g><text x="905.14" y="413.39" font-size="20" transform="rotate(270 905.14 413.39)" text-anchor="middle" dominant-baseline="middle">gopherjs/js — 5.2 KiB</text></g>
<g><rect x="835.87" y="810.95" width="138.54" height="120.95" fill="#53c7e0" fill-opacity="0.1" stroke="black" stroke-width="0.5"><title>internal/bytealg — 468 B own size, 810 B with children</title></rect></g>
<g></g>
<g><rect x="838.89" y="880.36" width="132.49" height="48.52" fill="#fb7d4c" fill-opacity="0.1" stroke="black" stroke-width="0.5"><title>internal/cpu — 342 B own size, 342 B with children</title></rect></g>
<g><text x="905.14" y="904.62" font-size="8.87" text-anchor="middle" dominant-baseline="middle">internal/cpu — 342 B</text></g>
<g><rect x="835.87" y="931.9" width="138.54" height="52.26" fill="#45ff00" fill-opacity="0.1" stroke="black" stroke-width="0.5"><title>runtime/internal/sys — 350 B own size, 350 B with children</title></rect></g>
<g></g>
<g><rect x="980.24" y="10" width="9.76" height="980" fill="#ff4dff" fill-opacity="0.1" stroke="black" stroke-width="0.5"><title>main — 434 B own size, 434 B with children</title></rect></g>
<g></g>
`
	got := b.String()
	fmt.Println(got)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("artifact.toSVG() returned diff (-want,+got):\n%s", diff)
	}
}
