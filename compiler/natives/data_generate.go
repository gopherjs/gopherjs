// +build generate

package main

import (
	"log"

	"github.com/gopherjs/gopherjs/compiler/natives"
	"github.com/shurcooL/vfsgen"
)

func main() {
	err := vfsgen.Generate(natives.Data, vfsgen.Options{
		PackageName:  "natives",
		BuildTags:    "!dev",
		VariableName: "Data",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
