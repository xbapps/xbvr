//go:build !windows
// +build !windows

package api

import "os/exec"

func buildCmd(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}
