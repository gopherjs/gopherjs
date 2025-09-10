//go:build js

package debug

import "time"

func setGCPercent(int32) int32 {
	// Not implemented. Return initial setting.
	return 100
}

func setMaxStack(bytes int) int {
	// Not implemented. Return initial setting.
	// The initial setting is 1 GB on 64-bit systems, 250 MB on 32-bit systems.
	return 250000000
}

func readGCStats(pauses *[]time.Duration) {
	// Not implemented. No GC stats available in this environment.
}

func freeOSMemory() {
	// Not implemented. No OS memory management in this environment.
}

func setPanicOnFault(bool) bool {
	// Not implemented.
	return true
}

func setMaxThreads(int) int {
	// Not implemented.
	return 10000
}

func setMemoryLimit(int64) int64 {
	// Not implemented.
	return 0
}
