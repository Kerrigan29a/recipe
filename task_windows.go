// +build windows

package recipe

import "os/exec"

func (t *Task) composeDefaultInterpreterCmd(spell string) []string {
	return []string{"cmd", "/c", spell}
}

func (t *Task) setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_UNICODE_ENVIRONMENT | syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
