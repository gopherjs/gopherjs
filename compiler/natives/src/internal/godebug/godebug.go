//go:build js
// +build js

package godebug

import (
	"sync"
)

type Setting struct {
	name string
	once sync.Once

	// temporarily replacement of atomic.Pointer[string] for go1.20 without generics.
	value *atomicStringPointer
}

type atomicStringPointer struct {
	v *string
}

func (x *atomicStringPointer) Load() *string     { return x.v }
func (x *atomicStringPointer) Store(val *string) { x.v = val }

func (s *Setting) Value() string {
	s.once.Do(func() {
		v, ok := cache.Load(s.name)
		if !ok {
			// temporarily replacement of atomic.Pointer[string] for go1.20 without generics.
			p := new(atomicStringPointer)
			p.Store(&empty)
			v, _ = cache.LoadOrStore(s.name, p)
		}
		// temporarily replacement of atomic.Pointer[string] for go1.20 without generics.
		s.value = v.(*atomicStringPointer)
	})
	return *s.value.Load()
}

// setUpdate is provided by package runtime.
// It calls update(def, env), where def is the default GODEBUG setting
// and env is the current value of the $GODEBUG environment variable.
// After that first call, the runtime calls update(def, env)
// again each time the environment variable changes
// (due to use of os.Setenv, for example).
//
// GOPHERJS: Currently we don't inject a proxy into process.env to watch
// for changes via syscall.runtimeSetenv and syscall.runtimeUnsetenv.
// We may want to look into this in the future.
func setUpdate(update func(string, string)) {}

func update(def, env string) {
	updateMu.Lock()
	defer updateMu.Unlock()

	did := make(map[string]bool)
	parse(did, env)
	parse(did, def)

	cache.Range(func(name, v any) bool {
		if !did[name.(string)] {
			// temporarily replacement of atomic.Pointer[string] for go1.20 without generics.
			v.(*atomicStringPointer).Store(&empty)
		}
		return true
	})
}

func parse(did map[string]bool, s string) {
	end := len(s)
	eq := -1
	for i := end - 1; i >= -1; i-- {
		if i == -1 || s[i] == ',' {
			if eq >= 0 {
				name, value := s[i+1:eq], s[eq+1:end]
				if !did[name] {
					did[name] = true
					v, ok := cache.Load(name)
					if !ok {
						// temporarily replacement of atomic.Pointer[string] for go1.20 without generics.
						p := new(atomicStringPointer)
						p.Store(&empty)
						v, _ = cache.LoadOrStore(name, p)
					}
					// temporarily replacement of atomic.Pointer[string] for go1.20 without generics.
					v.(*atomicStringPointer).Store(&value)
				}
			}
			eq = -1
			end = i
		} else if s[i] == '=' {
			eq = i
		}
	}
}
