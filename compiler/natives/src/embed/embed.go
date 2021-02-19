// +build js
// +build go1.16

package embed

func buildFS(list []struct {
	name string
	data string
	hash [16]byte
}) (f FS) {
	var files []file
	for _, v := range list {
		files = append(files, file{
			name: v.name,
			data: v.data,
			hash: v.hash,
		})
	}
	f.files = &files
	return
}
