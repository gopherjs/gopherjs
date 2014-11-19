// +build js

package time

func (l *Location) get() *Location {
	if l == nil {
		return &utcLoc
	}
	if l == &localLoc && localLoc.name == "" {
		initLocal()
	}
	return l
}
