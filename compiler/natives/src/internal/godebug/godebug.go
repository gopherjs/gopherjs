//go:build js
// +build js

package godebug

// setUpdate is provided by package runtime.
// It calls update(def, env), where def is the default GODEBUG setting
// and env is the current value of the $GODEBUG environment variable.
// After that first call, the runtime calls update(def, env)
// again each time the environment variable changes
// (due to use of os.Setenv, for example).
//
// GOPHERJS: For JS we currently will not be able
// to access $GODEBUG via process.env nor watch
// for changes via syscall.runtimeSetenv and
// syscall.runtimeUnsetenv
func setUpdate(update func(string, string)) {}
