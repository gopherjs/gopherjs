package analysis

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestRectSplit(t *testing.T) {
	tests := []struct {
		name     string
		original rect
		weights  []float64
		parts    rects
	}{
		{
			name:     "single part",
			original: rect{0, 1, 10, 2, true},
			weights:  []float64{42},
			parts:    rects{{0, 1, 10, 2, true}},
		},
		{
			name:     "two parts",
			original: rect{0, 1, 10, 2, true},
			weights:  []float64{4, 6},
			parts:    rects{{0, 1, 4, 2, true}, {4, 1, 10, 2, true}},
		},
		{
			name:     "many parts",
			original: rect{0, 1, 10, 2, true},
			weights:  []float64{2, 4, 6, 8},
			parts:    rects{{0, 1, 1, 2, true}, {1, 1, 3, 2, true}, {3, 1, 6, 2, true}, {6, 1, 10, 2, true}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.original.split(test.weights...)
			if diff := cmp.Diff(test.parts, got, cmp.AllowUnexported(rect{}), cmpopts.EquateApprox(0.0001, 0.0001)); diff != "" {
				t.Errorf("rect.split(%v) returned diff:\n%s", test.weights, diff)
			}
		})
	}
}

func TestRectToSVG(t *testing.T) {
	tests := []struct {
		name string
		rect rect
		opts []svgOpt
		want string
	}{
		{
			name: "default",
			rect: rect{left: 1, top: 2, right: 3, bottom: 4, transposed: false},
			want: `<g><rect x="1" y="2" width="2" height="2" fill="black" fill-opacity="1" stroke="black"></rect></g>`,
		}, {
			name: "transposed",
			rect: rect{left: 2, top: 1, right: 4, bottom: 3, transposed: true},
			want: `<g><rect x="1" y="2" width="2" height="2" fill="black" fill-opacity="1" stroke="black"></rect></g>`,
		}, {
			name: "red stroke",
			rect: rect{left: 1, top: 2, right: 3, bottom: 4, transposed: false},
			opts: []svgOpt{WithStroke("#F00", 0.5)},
			want: `<g><rect x="1" y="2" width="2" height="2" fill="black" fill-opacity="1" stroke="#F00" stroke-width="0.5"></rect></g>`,
		}, {
			name: "red fill",
			rect: rect{left: 1, top: 2, right: 3, bottom: 4, transposed: false},
			opts: []svgOpt{WithFill("#F00", 0.5)},
			want: `<g><rect x="1" y="2" width="2" height="2" fill="#F00" fill-opacity="0.5" stroke="black"></rect></g>`,
		}, {
			name: "with text",
			rect: rect{left: 10, top: 10, right: 80, bottom: 40, transposed: false},
			opts: []svgOpt{WithText("Hello, world!")},
			want: `<g><rect x="10" y="10" width="70" height="30" fill="black" fill-opacity="1" stroke="black"></rect>` +
				`<text x="45" y="25" font-size="8.08" text-anchor="middle" dominant-baseline="middle">Hello, world!</text></g>`,
		}, {
			name: "with text vertical",
			rect: rect{left: 10, top: 10, right: 40, bottom: 80, transposed: false},
			opts: []svgOpt{WithText("Hello, world!")},
			want: `<g><rect x="10" y="10" width="30" height="70" fill="black" fill-opacity="1" stroke="black"></rect>` +
				`<text x="25" y="45" font-size="8.08" transform="rotate(270 25.00 45.00)" text-anchor="middle" dominant-baseline="middle">Hello, world!</text></g>`,
		}, {
			name: "with text too small",
			rect: rect{left: 1, top: 1, right: 8, bottom: 4, transposed: false},
			opts: []svgOpt{WithText("Hello, world!")},
			want: `<g><rect x="1" y="1" width="7" height="3" fill="black" fill-opacity="1" stroke="black"></rect></g>`,
		}, {
			name: "with tooltip",
			rect: rect{left: 1, top: 2, right: 3, bottom: 4, transposed: false},
			opts: []svgOpt{WithTooltip("Hello, world!")},
			want: `<g><rect x="1" y="2" width="2" height="2" fill="black" fill-opacity="1" stroke="black"><title>Hello, world!</title></rect></g>`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b := &bytes.Buffer{}
			if err := test.rect.toSVG(b, test.opts...); err != nil {
				t.Fatalf("rect.toSVG() returned error: %s", err)
			}
			got := strings.TrimSpace(b.String())
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("rect.toSVG() returned diff (-want,+got):\n%s", diff)
			}
		})
	}
}
