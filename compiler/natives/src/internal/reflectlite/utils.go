package reflectlite

type ChanDir int

const (
	RecvDir ChanDir             = 1 << iota // <-chan
	SendDir                                 // chan<-
	BothDir = RecvDir | SendDir             // chan
)

type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}

var (
	ErrSyntax = &errorString{"invalid syntax"}
)

func unquote(s string) (string, error) {
	if len(s) < 2 {
		return s, nil
	}
	if s[0] == '\'' || s[0] == '"' {
		if s[len(s)-1] == s[0] {
			return s[1 : len(s)-1], nil
		}
		return "", ErrSyntax
	}
	return s, nil
}
