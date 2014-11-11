// +build js

package time

var localInitialized = false

func (l *Location) get() *Location {
	if l == nil {
		return &utcLoc
	}
	if l == &localLoc && !localInitialized {
		initLocal()
	}
	return l
}
