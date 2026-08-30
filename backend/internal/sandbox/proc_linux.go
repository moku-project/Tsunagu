//go:build linux

package sandbox

import (
	"os/exec"
	"syscall"
)

func childSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL, Setpgid: true}
}

func afterStart(*exec.Cmd) error { return nil }

func processAlive(cmd *exec.Cmd) bool {
	return cmd.Process.Signal(syscall.Signal(0)) == nil
}
