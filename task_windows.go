// +build windows

/*
 * Sources:
 * - https://github.com/mattn/goreman/blob/master/proc_windows.go
 * - https://github.com/golang/dep/issues/862
 * - https://github.com/alexbrainman/ps
 */


package recipe

import (
	"syscall"
	"os/exec"
	"fmt"
)

func (t *Task) composeDefaultInterpreterCmd(spell string) []string {
	return []string{"cmd", "/c", spell}
}

func (t *Task) setSysProcAttr() {
	t.cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_UNICODE_ENVIRONMENT | syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func (t *Task) Terminate() error {
	cmd := t.cmd
	if cmd == nil {
		return nil
	}
	p := cmd.Process
	if p == nil {
		return nil
	}

	// TODO: Use a better way. Probably using https://github.com/alexbrainman/ps
	// Search program
	path, err := exec.LookPath("taskkill")
	if err != nil {
		return err
	}
	err = exec.Command(path, "/F", "/T", "/PID", fmt.Sprint(p.Pid)).Run()
	if err != nil {
		return err
	}
	return nil
}
