// +build !windows

package recipe

func (t *Task) composeDefaultInterpreterCmd(spell string) []string {
	return []string{"/bin/sh", "-c", "exec " + spell}
}
