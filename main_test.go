package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"
	"testing"
)

func TestRun(t *testing.T) {
	testUTSNamespace()
}

// 隔离node name和domain name
func testUTSNamespace() {
	cmd := exec.Command("sh")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS,
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
