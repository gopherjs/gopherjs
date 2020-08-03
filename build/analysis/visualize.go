package analysis

import (
	"fmt"
	"os"

	"github.com/gopherjs/gopherjs/compiler"
)

// Visualizer renders a TreeMap diagram representing the generated JS file and
// how different packages contributed it its size.
type Visualizer struct {
	File *os.File
	Main *compiler.Archive
	Deps []*compiler.Archive
}

// Render a Tree Map diagram for the given package sizes.
//
// Stats contains amount of bytes written that correspond to a given package.
func (v *Visualizer) Render(stats map[string]int) error {
	tree := v.buildTree(stats)
	// This roughly corresponds to browser viewport dimensions on a 1080p screen.
	const w = 1500
	const h = 700
	fmt.Fprintf(v.File, `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">
<svg width='%d' height='%d' viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
<style> text { font-family: "Lucida Console", Monaco, monospace; text-rendering: optimizeLegibility; } </style>
`, w, h, w, h)
	if err := tree.toSVG(v.File, newRect(w, h)); err != nil {
		return err
	}
	fmt.Fprintf(v.File, "</svg>\n")
	return nil
}

func (v *Visualizer) buildTree(stats map[string]int) *treeMapNode {
	deps := map[string]*compiler.Archive{}

	for _, dep := range v.Deps {
		deps[dep.ImportPath] = dep
	}

	nodes := map[string]*treeMapNode{} // List of all nodes in the tree.
	roots := map[string]*treeMapNode{} // Nodes that are not someone's dependency.
	for name, size := range stats {
		nodes[name] = &treeMapNode{
			Label: name,
			Size:  float64(size),
		}
		roots[name] = nodes[name]
	}

	for name, node := range nodes {
		if dep, ok := deps[name]; ok {
			for _, depName := range dep.Imports {
				node.Children = append(node.Children, nodes[depName])
				delete(roots, depName)
			}
		}
	}

	topLevel := []*treeMapNode{}
	for _, node := range roots {
		topLevel = append(topLevel, node)
	}

	return (&treeMapNode{
		Label:    "$artifact",
		Size:     0, // No own size, just children.
		Children: topLevel,
		depth:    0,
	}).organizeTree()
}
