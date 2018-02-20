// +build !windows

package recipe

import (
	"os/exec"
	"syscall"
)

func (t *Task) composeDefaultInterpreterCmd(spell string) []string {
	return []string{"/bin/sh", "-c", "exec " + spell}
}

func (t *Task) setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
