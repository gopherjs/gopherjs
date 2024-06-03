package syscall

import (
	"syscall/js"
	_ "unsafe" // go:linkname
)

func runtime_envs() []string {
	process := js.Global().Get("process")
	if process.IsUndefined() {
		return nil
	}
	jsEnv := process.Get("env")
	if jsEnv.IsUndefined() {
		return nil
	}
	envkeys := js.Global().Get("Object").Call("keys", jsEnv)
	envs := make([]string, envkeys.Length())
	for i := 0; i < envkeys.Length(); i++ {
		key := envkeys.Index(i).String()
		envs[i] = key + "=" + jsEnv.Get(key).String()
	}
	return envs
}

func runtimeSetenv(k, v string) {
	setenv_c(k, v)
}

func runtimeUnsetenv(k string) {
	unsetenv_c(k)
}

func setenv_c(k, v string) {
	process := js.Global().Get("process")
	if process.IsUndefined() {
		return
	}
	process.Get("env").Set(k, v)
	godebug_notify(k, v)
}

func unsetenv_c(k string) {
	process := js.Global().Get("process")
	if process.IsUndefined() {
		return
	}
	process.Get("env").Delete(k)
	godebug_notify(k, ``)
}

//go:linkname godebug_notify runtime.godebug_notify
func godebug_notify(key, value string)

func setStat(st *Stat_t, jsSt js.Value) {
	// This method is an almost-exact copy of upstream, except for 4 places where
	// time stamps are obtained as floats in lieu of int64. Upstream wasm emulates
	// a 64-bit architecture and millisecond-based timestamps fit within an int
	// type. GopherJS is 32-bit and use of 32-bit ints causes timestamp truncation.
	// We get timestamps as float64 (which matches JS-native representation) and
	// convert then to int64 manually, since syscall/js.Value doesn't have an
	// Int64 method.
	st.Dev = int64(jsSt.Get("dev").Int())
	st.Ino = uint64(jsSt.Get("ino").Int())
	st.Mode = uint32(jsSt.Get("mode").Int())
	st.Nlink = uint32(jsSt.Get("nlink").Int())
	st.Uid = uint32(jsSt.Get("uid").Int())
	st.Gid = uint32(jsSt.Get("gid").Int())
	st.Rdev = int64(jsSt.Get("rdev").Int())
	st.Size = int64(jsSt.Get("size").Int())
	st.Blksize = int32(jsSt.Get("blksize").Int())
	st.Blocks = int32(jsSt.Get("blocks").Int())
	atime := int64(jsSt.Get("atimeMs").Float()) // Int64
	st.Atime = atime / 1000
	st.AtimeNsec = (atime % 1000) * 1000000
	mtime := int64(jsSt.Get("mtimeMs").Float()) // Int64
	st.Mtime = mtime / 1000
	st.MtimeNsec = (mtime % 1000) * 1000000
	ctime := int64(jsSt.Get("ctimeMs").Float()) // Int64
	st.Ctime = ctime / 1000
	st.CtimeNsec = (ctime % 1000) * 1000000
}

func Exit(code int) {
	if process := js.Global().Get("process"); !process.IsUndefined() {
		process.Call("exit", code)
		return
	}
	if code != 0 {
		js.Global().Get("console").Call("warn", "Go program exited with non-zero code:", code)
	}
}
