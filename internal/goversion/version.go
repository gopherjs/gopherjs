package goversion

import (
	"go/build"
	"strconv"

	"github.com/visualfc/goversion"
)

func ReleaseTags() []string {
	if installReleaseTags == nil {
		return build.Default.ReleaseTags
	}
	return installReleaseTags
}

var (
	installReleaseTags []string
)

func buildReleaseTags(version int) (tags []string) {
	for i := 1; i <= version; i++ {
		tags = append(tags, "go1."+strconv.Itoa(i))
	}
	return
}

func init() {
	ver, ok := goversion.Installed()
	if ok && ver.Major == 1 {
		installReleaseTags = buildReleaseTags(ver.Minor)
	} else {
		installReleaseTags = build.Default.ReleaseTags
	}
}
