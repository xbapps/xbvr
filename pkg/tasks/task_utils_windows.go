//go:build windows
// +build windows

package tasks

import (
	"os/exec"
	"syscall"
)

func buildCmd(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000 | 0x00004000, // CREATE_NO_WINDOW | BELOW_NORMAL_PRIORITY_CLASS
	}
	return cmd
}
