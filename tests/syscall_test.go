// +build js

package tests

import (
	"io/ioutil"
	"os"
	"syscall"
	"testing"
)

func TestGetpid(t *testing.T) {
	pid := syscall.Getpid()
	if pid <= 0 {
		t.Errorf("Got invalid pid %d. Want: > 0", pid)
	} else {
		t.Logf("Got pid %d", pid)
	}
}

func TestOpen(t *testing.T) {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Failed to create a temp file: %s", err)
	}
	f.Close()
	defer os.Remove(f.Name())
	fd, err := syscall.Open(f.Name(), syscall.O_RDONLY, 0600)
	if err != nil {
		t.Fatalf("syscall.Open() returned error: %s", err)
	}
	err = syscall.Close(fd)
	if err != nil {
		t.Fatalf("syscall.Close() returned error: %s", err)
	}
}
