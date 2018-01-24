//go:generate stringer -type=TaskState

package recipe

import (
	"os"
	"os/exec"
	"errors"
	"context"
	"sync"
	"encoding/json"
	"bytes"
	"runtime"
	"strings"
)

/***
* Sources:
*  - https://blog.kowalczyk.info/article/wOYk/advanced-command-execution-in-go-with-osexec.html
*/

type TaskState int

const (
	Disabled TaskState = iota
	Enabled
	Waiting
	Running
	Success
	Failure
)

type Task struct {
	Deps         []string          `json:"deps" toml:"deps"`
	Env          map[string]string `json:"env" toml:"env"`
	Interp       []string          `json:"interp" toml:"interp"`
	Cmd          string            `json:"cmd" toml:"cmd"`
	Stdout       string            `json:"stdout" toml:"stdout"`
	Stderr       string            `json:"stderr" toml:"stderr"`
	AllowFailure bool              `json:"allow_failure" toml:"allow_failure"`
	State        TaskState         `json:"state" toml:"state"`
	cancel       context.CancelFunc
	mu           sync.RWMutex
}

/***
* Task
*/

func (t *Task) composeEnv(r *Recipe) []string {
	newEnv := os.Environ()
	for key, value := range r.Environ() {
		newEnv = append(newEnv, key+"="+value)
	}
	for key, value := range t.Environ() {
		newEnv = append(newEnv, key+"="+value)
	}
	return newEnv
}

func replaceCmd(parts []string, spell string) []string {
	newParts := make([]string, len(parts))
	for i, txt := range parts {
		newParts[i] = strings.Replace(txt, "{cmd}", spell, -1)
	}
	return newParts
}

func (t *Task) composeInterpreterCmd(spell string, r *Recipe) []string {
	// Check task config
	if parts := t.Interpreter(); parts != nil {
		if len(parts) == 0 {
			goto defaultConfig
		}
		return replaceCmd(parts, spell)
	}
	// Check recipe config
	if parts := r.Interpreter(); parts != nil {
		if len(parts) == 0 {
			goto defaultConfig
		}
		return replaceCmd(parts, spell)
	}
defaultConfig:
	// Default config
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/c", spell}
	}
	return []string{"/bin/sh", "-c", "exec " + spell}
}

func (t *Task) Execute(ctx context.Context, r *Recipe) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	parts := t.composeInterpreterCmd(t.Cmd, r)
	env := t.composeEnv(r)

	// Search program
	path, err := exec.LookPath(parts[0])
	if err != nil {
		return err
	}
	// Create cmd
	cmd := exec.CommandContext(ctx, path, parts[1:]...)
	// Redirect stdout and stderr
	if t.Stdout != "" {
		f, err := os.Create(t.Stdout)
		if err != nil {
			return err
		}
		defer f.Close()
		cmd.Stdout = f
	} else {
		cmd.Stdout = os.Stdout
	}
	if t.Stderr != "" {
		f, err := os.Create(t.Stderr)
		if err != nil {
			return err
		}
		defer f.Close()
		cmd.Stderr = f
	} else {
		cmd.Stderr = os.Stderr
	}
	cmd.Env = env
	// Run
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func (t *Task) SetEnabled() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.State != Disabled {
		panic(errors.New("Current state must be Disabled"))
	}
	t.State = Enabled
}

func (t *Task) IsEnabled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.State == Enabled
}

func (t *Task) SetWaiting() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.State != Enabled {
		panic(errors.New("Current state must be Enabled"))
	}
	t.State = Waiting
}

func (t *Task) IsWaiting() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.State == Waiting
}

func (t *Task) SetRunning() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.State != Waiting {
		panic(errors.New("Current state must be Waiting"))
	}
	t.State = Running
}

func (t *Task) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.State == Running
}

func (t *Task) SetSuccess() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.State != Running {
		panic(errors.New("Current state must be Running"))
	}
	t.State = Success
}

func (t *Task) IsSuccess() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.State == Success
}

func (t *Task) SetFailure() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.State != Running {
		panic(errors.New("Current state must be Running"))
	}
	t.State = Failure
}

func (t *Task) IsFailure() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.State == Failure
}

func (t *Task) SetCancel(cancel context.CancelFunc) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cancel = cancel
}

func (t *Task) Cancel() context.CancelFunc {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.cancel
}

func (t *Task) Environ() map[string]string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Env
}

func (t *Task) Interpreter() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.Interp
}

func (t *Task) String() string {
	return t.string(false)
}

func (t *Task) PrettyString() string {
	return t.string(true)
}

func (t *Task) string(indent bool) string {
	b := bytes.Buffer{}
	e := json.NewEncoder(&b)
	if indent {
		e.SetIndent("", " ")
	}
	err := e.Encode(t)
	if err != nil {
		panic(err)
	}
	return b.String()
}
