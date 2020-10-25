// +build js

package time

func initTestingZone() {
	z, err := loadLocation("America/Los_Angeles", zoneSources)
	if err != nil {
		panic("cannot load America/Los_Angeles for testing: " + err.Error())
	}
	z.name = "Local"
	localLoc = *z
}

func forceZipFileForTesting(zipOnly bool) {
}
