//go:build js

package embed

//gopherjs:new used by gopherjs/build/embed.go
func buildFS(list []struct {
	name string
	data string
	hash [16]byte
},
) (f FS) {
	n := len(list)
	files := make([]file, n)
	for i := 0; i < n; i++ {
		files[i].name = list[i].name
		files[i].data = list[i].data
		files[i].hash = list[i].hash
	}
	f.files = &files
	return
}
