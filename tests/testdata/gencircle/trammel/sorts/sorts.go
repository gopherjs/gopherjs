package sorts

import "github.com/gopherjs/gopherjs/tests/testdata/gencircle/trammel/cmp"

func Pair[K cmp.Ordered, V any, SK ~[]K, SV ~[]V](k SK, v SV) {
	Bubble(struct { // non-generic struct in a generic context.
		len  func() int
		less func(i, j int) bool
		swap func(i, j int)
	}{
		len:  func() int { return len(k) },
		less: func(i, j int) bool { return k[i] < k[j] },
		swap: func(i, j int) { k[i], v[i], k[j], v[j] = k[j], v[j], k[i], v[i] },
	})
}

func Bubble(f struct {
	len  func() int
	less func(i, j int) bool
	swap func(i, j int)
}) {
	length := f.len()
	for i := 0; i < length; i++ {
		for j := i + 1; j < length; j++ {
			if f.less(j, i) {
				f.swap(i, j)
			}
		}
	}
}
