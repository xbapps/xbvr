//go:build windows
// +build windows

package api

import (
	"os/exec"
	"syscall"
)

func buildCmd(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}
