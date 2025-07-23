package cat

import "github.com/gopherjs/gopherjs/tests/testdata/gencircle/pingpong/collections"

type Cat[H collections.Hasher] struct {
	Name string
}

func (c Cat[H]) Hash() uint {
	var zero H
	var h collections.Hasher = zero
	for _, v := range c.Name {
		h = h.Add(uint(v))
	}
	return h.Sum()
}
