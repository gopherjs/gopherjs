package syscall

import (
	"github.com/gopherjs/gopherjs/js"
)

// FIXME(nevkontakte): Duplicated from syscall_unix.go.
func runtime_envs() []string {
	process := js.Global.Get("process")
	if process == js.Undefined {
		return nil
	}
	jsEnv := process.Get("env")
	envkeys := js.Global.Get("Object").Call("keys", jsEnv)
	envs := make([]string, envkeys.Length())
	for i := 0; i < envkeys.Length(); i++ {
		key := envkeys.Index(i).String()
		envs[i] = key + "=" + jsEnv.Get(key).String()
	}
	return envs
}

func setenv_c(k, v string) {
	process := js.Global.Get("process")
	if process == js.Undefined {
		return
	}
	process.Get("env").Set(k, v)
}

func unsetenv_c(k string) {
	process := js.Global.Get("process")
	if process == js.Undefined {
		return
	}
	process.Get("env").Delete(k)
}
