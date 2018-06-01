// +build !windows

/*
 * Sources:
 * - https://github.com/mattn/goreman/blob/master/proc_posix.go
 * - https://github.com/golang/dep/issues/862
 */

package recipe

import (
	"os"
	"syscall"
)

func (t *Task) composeDefaultInterpreterCmd(spell string) []string {
	return []string{"/bin/sh", "-c", "exec " + spell}
}

func (t *Task) setSysProcAttr() {
	t.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
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

	pgid, err := syscall.Getpgid(p.Pid)
	if err != nil {
		return err
	}

	// Use pgid
	// From: http://unix.stackexchange.com/questions/14815/process-descendants
	pid := p.Pid
	if pgid == p.Pid {
		pid = -1 * pid
	}

	target, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	//return target.Signal(syscall.SIGHUP)
	return target.Signal(syscall.SIGKILL)
	//return target.Kill()
}
