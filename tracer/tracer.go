package tracer

import (
	"fmt"
	"strings"
	"sync"
)

var global = tracer{}

func Tracef(format string, args ...interface{}) {
	global.Tracef(format, args...)
}

func Scope(format string, args ...interface{}) scope {
	return global.Scope(format, args...)
}

type scope struct {
	t    *tracer
	name string
}

func (s scope) String() string { return s.name }

func (s scope) Leave(e *error) {
	s.t.mu.Lock()
	defer s.t.mu.Unlock()

	top := s.t.stack[0]
	if top.name != s.name {
		s.t.Tracef("WARNING: Leaving scope %q, when expected to leave %q.", top, s)
	}
	oldStack := s.t.stack
	s.t.stack = s.t.stack[1:]
	s.t.traceInternal("<< Leaving %s...", oldStack)
	if e != nil && *e != nil {
		s.t.traceInternal("   ! with error: %s", *e)
	}
}

type scopes []scope

func (ss scopes) String() string {
	names := []string{}
	for _, s := range ss {
		names = append(names, s.name)
	}
	return strings.Join(names, " → ")
}

type tracer struct {
	mu    sync.Mutex
	stack scopes
}

func (t *tracer) Tracef(format string, args ...interface{}) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.traceInternal(format, args...)
}

func (t *tracer) traceInternal(format string, args ...interface{}) {
	fmt.Printf(strings.Repeat("   ", len(t.stack))+format+"\n", args...)
}

func (t *tracer) Scope(format string, args ...interface{}) scope {
	t.mu.Lock()
	defer t.mu.Unlock()
	s := scope{t: t, name: fmt.Sprintf(format, args...)}
	newStack := append(scopes{s}, t.stack...)
	t.traceInternal(">> Entering %s...", newStack)
	t.stack = newStack
	return s
}
