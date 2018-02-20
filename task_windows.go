// +build windows

// From: https://github.com/mattn/goreman/blob/master/proc_windows.go

package recipe

import (
	"syscall"
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
	kernel32, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return err
	}
	defer kernel32.Release()

	pid := t.cmd.Process.Pid

	setConsoleCtrlHandler, err := kernel32.FindProc("SetConsoleCtrlHandler")
	if err != nil {
		return err
	}
	result, _, err := setConsoleCtrlHandler.Call(0, 1)
	if result == 0 {
		return err
	}
	generateConsoleCtrlEvent, err = kernel32.FindProc("GenerateConsoleCtrlEvent")
	if err != nil {
		return err
	}
	result, _, err = generateConsoleCtrlEvent.Call(syscall.CTRL_BREAK_EVENT, uintptr(pid))
	if result == 0 {
		return err
	}
	return nil
}
