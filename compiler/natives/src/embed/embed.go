// +build js
// +build go1.16

package embed

func appendData(f *FS, name string, data string, hash [16]byte) {
	var files []file
	if f.files != nil {
		files = *f.files
	}
	files = append(files, file{
		name: name,
		data: data,
		hash: hash,
	})
	f.files = &files
}
