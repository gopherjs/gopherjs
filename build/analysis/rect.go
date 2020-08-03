package analysis

import (
	"encoding/xml"
	"fmt"
	"io"
	"math"
)

// rect represents a single rectangle area on the SVG rendering of the TreeMap diagram.
type rect struct {
	left       float64
	top        float64
	right      float64
	bottom     float64
	transposed bool
}

func newRect(w, h float64) rect { return rect{0, 0, w, h, false} }

func (r rect) width() float64 { return r.right - r.left }

func (r rect) height() float64 { return r.bottom - r.top }

// long returns the length of the longer side of the rectangle.
func (r rect) long() float64 {
	w, h := r.width(), r.height()
	if w > h {
		return w
	}
	return h
}

// short returns the length of the shorter side of the rectangle.
func (r rect) short() float64 {
	w, h := r.width(), r.height()
	if w < h {
		return w
	}
	return h
}

// padding between the current rect's bounds and nested rectangles representing
// child nodes in the TreeMap.
func (r rect) padding() float64 {
	const factor = 0.025
	padding := r.short() * factor
	if padding > 10 {
		padding = 10
	}
	return padding
}

// shrink the rectangle bounds by the given margin on every side.
//
// The center of the shrinked rectangle will be the same. Padding is
// automatically reduced to prevent shrinking to zero if necessary.
func (r rect) shrink(padding float64) rect {
	if padding*2 > r.short()/2 {
		padding = r.short() / 4
	}
	shrinked := rect{
		left:       r.left + padding,
		top:        r.top + padding,
		right:      r.right - padding,
		bottom:     r.bottom - padding,
		transposed: r.transposed,
	}
	return shrinked
}

// aspectRatio returns width divided by height.
func (r rect) aspectRatio() float64 { return r.width() / r.height() }

// transpose flips X and Y coordinates of the rectangle and sets the
// "transposed" flag appropriately.
func (r rect) transpose() rect {
	return rect{
		left:       r.top,
		top:        r.left,
		right:      r.bottom,
		bottom:     r.right,
		transposed: !r.transposed,
	}
}

// orientHorizontally transposes the rectangle such that the horizontal size is
// always the longer.
//
// When subdividing a rectangle in the diagram we always lay down stacks of
// children along the longer side for better readability, always orienting the
// rectangle horizontally prevents a lot of branching in the layout code.
// Calling restoreOrientation() will place the rectangle in the corrent place
// regardless of how many times it or its parents were transposed.
func (r rect) orientHorizontally() rect {
	if r.aspectRatio() < 1 {
		return r.transpose()
	}
	return r
}

// restoreOrientation transposes the rectangle if necessary to return it to the
// original coordinate system.
func (r rect) restoreOrientation() rect {
	if r.transposed {
		return r.transpose()
	}
	return r
}

// split the rectangle along the horizontal axis in protortion to the weights.
// transpose() the rectangle first in order to split along the vertical axis.
func (r rect) split(weights ...float64) rects {
	var total float64
	for _, w := range weights {
		total += w
	}

	parts := []rect{}
	var processedFraction float64
	for _, w := range weights {
		fraction := w / total
		parts = append(parts, rect{
			left:       r.left + r.width()*processedFraction,
			top:        r.top,
			right:      r.left + r.width()*(processedFraction+fraction),
			bottom:     r.bottom,
			transposed: r.transposed,
		})
		processedFraction += fraction
	}

	return parts
}

// toSVG writes the SVG code to represent the current rect given the display options.
//
// Coordinates and sizes are rounded to 2 decimal digits to reduce chances of
// tests flaking out due to floating point imprecision. This will have no
// visible impact on the rendering since 0.01 unit == 0.01 pixel by default.
func (r rect) toSVG(w io.Writer, opts ...svgOpt) error {
	normal := r.restoreOrientation() // Make sure we render the rect in its actual position.

	// Populate the default SVG representation of the rect for its coordinates.
	data := svgGroup{
		Rect: &svgRect{
			X:           round2(normal.left),
			Y:           round2(normal.top),
			Width:       round2(normal.width()),
			Height:      round2(normal.height()),
			Fill:        "black",
			FillOpacity: 1,
			Stroke:      "black",
			StrokeWidth: 0,
		},
		Text: &svgText{
			X:          round2(normal.left + normal.width()/2),
			Y:          round2(normal.top + normal.height()/2),
			TextAnchor: "middle",
			Baseline:   "middle",
			FontSize:   20,
		},
	}

	// Apply all the display options passed by the caller.
	for _, o := range opts {
		o(&data)
	}

	// Adjust rect label display such that it is readable and fits rectangle
	// bounds. The constants below are empirically determined.
	const (
		fontFactorX = 1.5
		fontFactorY = 0.8
		minFont     = 8
	)
	if data.Text.FontSize > normal.short()*fontFactorY {
		data.Text.FontSize = normal.short() * fontFactorY
	}
	if l := float64(len(data.Text.Text)); l*data.Text.FontSize > normal.long()*fontFactorX {
		data.Text.FontSize = normal.long() / l * fontFactorX
	}
	if data.Text.FontSize < minFont {
		data.Text.Text = ""
	}
	data.Text.FontSize = round2(data.Text.FontSize)
	if normal.aspectRatio() < 1 {
		data.Text.Transform = fmt.Sprintf("rotate(270 %0.2f %0.2f)", data.Text.X, data.Text.Y)
	}

	if data.Text.Text == "" {
		// Remove the text element if there's nothing to show.
		data.Text = nil
	}
	if data.Rect.FillOpacity == 0 && data.Rect.StrokeWidth == 0 {
		// Remove the rect element if it isn't supposed to be visible.
		data.Rect = nil
	}

	defer w.Write([]byte("\n"))
	return xml.NewEncoder(w).Encode(data)
}

type svgGroup struct {
	XMLName struct{} `xml:"g"`
	Rect    *svgRect
	Text    *svgText `xml:",omitempty"`
}

type svgRect struct {
	XMLName     struct{} `xml:"rect"`
	X           float64  `xml:"x,attr"`
	Y           float64  `xml:"y,attr"`
	Width       float64  `xml:"width,attr"`
	Height      float64  `xml:"height,attr"`
	Fill        string   `xml:"fill,attr,omitempty"`
	FillOpacity float64  `xml:"fill-opacity,attr"`
	Stroke      string   `xml:"stroke,attr,omitempty"`
	StrokeWidth float64  `xml:"stroke-width,attr,omitempty"`
	Title       string   `xml:"title,omitempty"`
}

type svgText struct {
	XMLName    struct{} `xml:"text"`
	Text       string   `xml:",chardata"`
	X          float64  `xml:"x,attr"`
	Y          float64  `xml:"y,attr"`
	FontSize   float64  `xml:"font-size,attr"`
	Transform  string   `xml:"transform,attr,omitempty"`
	TextAnchor string   `xml:"text-anchor,attr,omitempty"`
	Baseline   string   `xml:"dominant-baseline,attr,omitempty"`
}

type svgOpt func(r *svgGroup)

// WithTitle adds a hover tooltip text to the rectangle.
func WithTooltip(t string) svgOpt {
	return func(g *svgGroup) { g.Rect.Title = t }
}

// WithText adds a text label over the rectangle.
func WithText(t string) svgOpt {
	return func(g *svgGroup) { g.Text.Text = t }
}

// WithFill sets rectangle fill style.
func WithFill(color string, opacity float64) svgOpt {
	return func(g *svgGroup) {
		g.Rect.Fill = color
		g.Rect.FillOpacity = opacity
	}
}

// WithStroke sets rectangle outline stroke style.
func WithStroke(color string, width float64) svgOpt {
	return func(g *svgGroup) {
		g.Rect.Stroke = color
		g.Rect.StrokeWidth = width
	}
}

// rects is a group of rectangles representing sibling nodes in the tree.
type rects []rect

// maxAspect returns the highest aspect ratio amount the rect group.
//
// Aspect ratios lesser than 1 are inverted to be above 1. The closer the return
// value is to 1, the closer all rectangles in the group are to squares.
func (rr rects) maxAspect() float64 {
	var result float64 = 1 // Start as if we have a perfectly square layout.
	for _, r := range rr {
		aspect := r.aspectRatio()
		if aspect < 1 {
			aspect = 1 / aspect
		}
		if aspect > result {
			result = aspect
		}
	}
	return result
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
