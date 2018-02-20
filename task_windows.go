// +build windows

package recipe

func (t *Task) composeDefaultInterpreterCmd(spell string) []string {
	return []string{"cmd", "/c", spell}
}
