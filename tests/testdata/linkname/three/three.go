package three

func DoThree() string {
	return "doing three"
}

func init() {
	// Avoid dead-code elimination.
	// TODO(nevkontakte): This should not be necessary.
	var _ = doInternalThree
}

var threeSecret = "three secret"

// This function is unexported and can't be accessed by other packages via a
// conventional import.
func doInternalThree() string {
	return "doing internal three: " + threeSecret
}
