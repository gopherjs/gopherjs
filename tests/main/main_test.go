package main

import "testing"

func TestMain(t *testing.T) {
	if mainDidRun {
		t.Error("main function did run")
	}
}
