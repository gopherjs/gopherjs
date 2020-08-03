package analysis

import (
	"crypto/md5"
	"fmt"
	"image/color"
	"io"
	"math"
	"math/bits"
	"sort"
	"strings"

	"github.com/dustin/go-humanize"
)

// treeMapNode represents a single node in a Tree Map diagram.
//
// Each node has a label and own size associated with it, as well as a list of
// direct descendants.
//
// Even though this type can be used for building Tree Map diagrams of anything
// (e.g. file system) it has extra features to handle dependency graph use case.
// Node's children must form a directed acyclic graph, not necessarily a tree.
// It will be later transformed into a tree by a organizeTree() call.
type treeMapNode struct {
	Label    string
	Size     float64
	Children nodeGroup

	// The following fields are populated from the tree structure by organizeTree()
	depth             int          // Shortest distance between the node and the tree root.
	parent            *treeMapNode // Selected tree parent for the current node.
	effectiveChildren nodeGroup    // Nodes for which this node was selected as a parent sorted by cumulative size.
	cumulativeSize    float64
}

// organizeTree transforms the acyclic import graph into a spanning tree.
//
// In order to visualize import graph and package sizes as a Tree Map diagram we
// need to reduce it to a tree to make sure we don't double-count size
// contribution of transitive dependencies even if they are imported by multiple
// other packages. We solve this by selecting a single parent for every node of
// the import graph and rendering the node only under the selected parent.
// Empirically, the diagram is most readable if we selected the parent with the
// shortest distance from the tree root (main package).
func (root *treeMapNode) organizeTree() *treeMapNode {
	// Mark all nodes as initially not visited.
	var prepare func(node *treeMapNode)
	prepare = func(node *treeMapNode) {
		node.depth = 1<<(bits.UintSize-1) - 1 // Max int
		for _, c := range node.Children {
			prepare(c)
		}
	}
	prepare(root)
	root.depth = 0

	// Use Dijkstra's algorithm to build a spanning tree.
	queue := []*treeMapNode{root}
	processNode := func(node *treeMapNode) {
		// Sort children by name to ensure deterministic result.
		sort.Slice(node.Children, func(i, j int) bool {
			return node.Children[i].Label < node.Children[j].Label
		})
		// Select provisional parents for all children.
		for _, child := range node.Children {
			if child.depth > node.depth+1 {
				child.depth = node.depth + 1
				child.parent = node
			}
			queue = append(queue, child)
		}
	}
	for len(queue) > 0 {
		processNode(queue[0])
		queue = queue[1:]
	}

	// Precompute some numbers we'll been to build the diagram.
	var precompute func(node *treeMapNode)
	precompute = func(node *treeMapNode) {
		node.cumulativeSize = node.Size
		for _, child := range node.Children {
			if child.parent == node {
				precompute(child) // Compute all the metrics for the child first.
				node.effectiveChildren = append(node.effectiveChildren, child)
				node.cumulativeSize += child.cumulativeSize
			}
		}
		// Sort effective children by size descending, this tends to produce better-looking diagrams.
		sort.Slice(node.effectiveChildren, func(i, j int) bool {
			if node.effectiveChildren[i].cumulativeSize != node.effectiveChildren[j].cumulativeSize {
				return node.effectiveChildren[i].cumulativeSize > node.effectiveChildren[j].cumulativeSize
			}
			return node.effectiveChildren[i].Label < node.effectiveChildren[j].Label // To guarantee determinism.
		})
	}
	precompute(root)
	return root
}

// color assigns a deterministic pseudo-random color to the node. Colors are in
// light pastel gamma to provide good readable contrast for text labels.
func (n *treeMapNode) color() string {
	hash := md5.Sum([]byte(n.Label))
	cb := hash[0]
	cr := hash[1]
	y := uint8(256 * (0.25 + math.Pow(0.9, float64(n.depth))/2))
	r, g, b := color.YCbCrToRGB(y, cb, cr)
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// renderSelf renders a labeled rectangle for the node itself within the given
// bounding area and returns a sub-area where children nodes should be rendered.
// Bounding area is split between self and children based on the relative
// weights.
func (n *treeMapNode) renderSelf(w io.Writer, bounds rect) (rect, error) {
	// Output own bounding rect into the SVG stream.
	if err := bounds.toSVG(w, WithTooltip(n.Describe()), WithFill(n.color(), 0.1), WithStroke("black", 0.5)); err != nil {
		return rect{}, fmt.Errorf("failed to render self to svg: %s", err)
	}

	// Create a bit of a border for better visualization of the tree hierarchy.
	bounds = bounds.shrink(bounds.padding())

	// Reserve some space to represent own size.
	ownSplit := bounds.split(n.Size, n.cumulativeSize-n.Size)
	ownSplit[0].toSVG(w, WithFill("none", 0), WithText(fmt.Sprintf("%s — %s", n.Label, humanize.IBytes(uint64(n.Size)))))

	// The rest will be allocated to children.
	bounds = ownSplit[1]
	return bounds, nil
}

// toSVG renders Tree Map diagram for the node and its children in the given bounding area.
//
// This function implements a simplified version of the Squarified Treemaps [1]
// algorithm to avoid generating a lot of thin and long nodes, which are
// difficult to read and label.
//
// Note that organizeTree() needs to be called prior to this method to ensure
// every element in the graph is rendered exactly once.
//
// [1]: Bruls, Mark, Kees Huizing, and Jarke J. Van Wijk. "Squarified treemaps."
// Data visualization 2000. Springer, Vienna, 2000. 33-42.
func (n *treeMapNode) toSVG(w io.Writer, bounds rect) error {
	// Reserve a fraction of the bounding area to represent node's own weight.
	bounds, err := n.renderSelf(w, bounds)
	if err != nil {
		return fmt.Errorf("failed to render self: %s", err)
	}

	if len(n.effectiveChildren) == 0 {
		return nil
	}

	// Renders a stack of child nodes within the corresponding bounding areas.
	renderStack := func(stack nodeGroup, bounds rects) error {
		if len(stack) != len(bounds) {
			// This should never happen ™.
			panic(fmt.Sprintf("Tried to lay out node stack %v inside %d bounding rects, need %d", stack, len(bounds), len(stack)))
		}
		for i, node := range stack {
			node.toSVG(w, bounds[i])
		}
		return nil
	}

	bounds = bounds.orientHorizontally()
	stack := nodeGroup{n.effectiveChildren[0]}                  // Seed the first stack of children.
	queue := n.effectiveChildren[1:]                            // Queue the remaining children.
	split := bounds.split(stack.totalSize(), queue.totalSize()) // Split the available space between the stack and the remainder.
	stackBounds := split[0].split(stack.cumulativeSizes()...)   // Subdivide stack space for each stack element.

	// Attempt to arrange children into one or more "stacks" inside the bounding
	// area such that each child's area is as square as possible. We do it
	// greedily by adding children into stacks as long as it improves aspect
	// ratios within the stack.
	for len(queue) > 0 {
		var candidate *treeMapNode
		candidate, queue = queue[0], queue[1:]
		newStack := append(stack, candidate)
		newSplit := bounds.split(newStack.totalSize(), queue.totalSize())
		newStackBounds := newSplit[0].orientHorizontally().split(newStack.cumulativeSizes()...)

		if newStackBounds.maxAspect() > stackBounds.maxAspect() {
			// The new stack made the layout less square, so we commit the the
			// previous version of the stack and seed a new stack.
			if err := renderStack(stack, stackBounds); err != nil {
				return fmt.Errorf("failed to lay out children of %q: %s", n.Label, err)
			}

			bounds = split[1]            // Reduce the available layout space to that's left.
			stack = nodeGroup{candidate} // Seed the new stack.
			split = bounds.split(stack.totalSize(), queue.totalSize())
			stackBounds = split[0].split(stack.cumulativeSizes()...)
		} else {
			// The new stack looks better, accept it and keep adding to it.
			stack = newStack
			split = newSplit
			stackBounds = newStackBounds
		}
	}

	// Render the last stack as it is.
	if err := renderStack(stack, stackBounds); err != nil {
		return fmt.Errorf("failed to lay out children of %q: %s", n.Label, err)
	}

	return nil
}

// Describe the node itself.
func (n *treeMapNode) Describe() string {
	return fmt.Sprintf("%s — %s own size, %s with children", n.Label, humanize.IBytes(uint64(n.Size)), humanize.IBytes(uint64(n.cumulativeSize)))
}

// String representation of the node and its effective children.
func (n *treeMapNode) String() string {
	s := &strings.Builder{}
	fmt.Fprintf(s, " %s %s\n", strings.Repeat(" *", n.depth), n.Describe())
	for _, c := range n.effectiveChildren {
		s.WriteString(c.String())
	}
	return s.String()
}

// nodeGroup represents a group of adjacent tree nodes.
type nodeGroup []*treeMapNode

func (ng nodeGroup) totalSize() float64 {
	var size float64
	for _, c := range ng {
		size += c.cumulativeSize
	}
	return size
}

func (ng nodeGroup) cumulativeSizes() []float64 {
	sizes := []float64{}
	for _, c := range ng {
		sizes = append(sizes, c.cumulativeSize)
	}
	return sizes
}
