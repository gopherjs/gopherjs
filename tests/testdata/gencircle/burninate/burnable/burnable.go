package burnable

import "github.com/gopherjs/gopherjs/tests/testdata/gencircle/burninate/dragon"

type Cottages struct{}

func (c Cottages) String() string {
	return `thatched-roof cottages`
}

func Burn[B dragon.Burnable](d dragon.Trogdor[B], b B) {
	d.Burninate(b)
}
