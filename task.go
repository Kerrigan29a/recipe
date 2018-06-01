package recipe

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"sync"
)

/*
 * Sources:
 * - https://blog.kowalczyk.info/article/wOYk/advanced-command-execution-in-go-with-osexec.html
 */

type Task struct {
	Deps         []string          `json:"deps" toml:"deps"`
	Env          map[string]string `json:"env" toml:"env"`
	Interp       []string          `json:"interp" toml:"interp"`
	Cmd          string            `json:"cmd" toml:"cmd"`
	Stdout       string            `json:"stdout" toml:"stdout"`
	Stderr       string            `json:"stderr" toml:"stderr"`
	AllowFailure bool              `json:"allow_failure" toml:"allow_failure"`
	cmd          *exec.Cmd
	mu           sync.RWMutex
}

/***
 * Task
 */

/*

// NOTE: This snippet shows how to reset unmarshaled structs to have different default values
// WARNING: At this moment is not working with TOML due to a go-toml limitation: https://github.com/pelletier/go-toml/blob/master/marshal.go#L318

// Used to avoid recursion in UnmarshalJSON below.
type task Task

func (t *task) reset() {
	t.Deps = make([]string, 0)
	t.Env = make(map[string]string, 0)
	t.Interp = make([]string, 0)
	t.Cmd = ""
	t.Stdout = ""
	t.Stderr = ""
	t.AllowFailure = false
}

func (t *Task) UnmarshalJSON(b []byte) error {
	fmt.Printf("DEBUG TASK UnmarshalJSON\n")

	newT := task{}
	newT.reset()

	err := json.Unmarshal(b, &newT)
	if err != nil {
		return err
	}

	t.Deps = newT.Deps
	t.Env = newT.Env
	t.Interp = newT.Interp
	t.Cmd = newT.Cmd
	t.Stdout = newT.Stdout
	t.Stderr = newT.Stderr
	t.AllowFailure = newT.AllowFailure
	t.mu = newT.mu

	fmt.Printf("DEBUG task = %s\n", t)
	return nil
}

func (t *Task) UnmarshalTOML(b []byte) error {
	fmt.Printf("DEBUG TASK UnmarshalTOML\n")

	newT := task{}
	newT.reset()

	err := toml.Unmarshal(b, &newT)
	if err != nil {
		return err
	}

	t.Deps = newT.Deps
	t.Env = newT.Env
	t.Interp = newT.Interp
	t.Cmd = newT.Cmd
	t.Stdout = newT.Stdout
	t.Stderr = newT.Stderr
	t.AllowFailure = newT.AllowFailure
	t.mu = newT.mu

	fmt.Printf("DEBUG task = %s\n", t)
	return nil
}

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
			return t.composeDefaultInterpreterCmd(spell)
		}
		return replaceCmd(parts, spell)
	}
	// Check recipe config
	if parts := r.Interpreter(); parts != nil {
		if len(parts) == 0 {
			return t.composeDefaultInterpreterCmd(spell)
		}
		return replaceCmd(parts, spell)
	}
	return t.composeDefaultInterpreterCmd(spell)
}

func (t *Task) Execute(r *Recipe) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.Cmd == "" {
		return nil
	}

	parts := t.composeInterpreterCmd(t.Cmd, r)
	env := t.composeEnv(r)

	// Search program
	path, err := exec.LookPath(parts[0])
	if err != nil {
		return err
	}
	// Create cmd
	t.cmd = exec.Command(path, parts[1:]...)
	// Redirect stdout and stderr
	if t.Stdout != "" {
		f, err := os.Create(t.Stdout)
		if err != nil {
			return err
		}
		defer f.Close()
		t.cmd.Stdout = f
	} else {
		t.cmd.Stdout = os.Stdout
	}
	if t.Stderr != "" {
		f, err := os.Create(t.Stderr)
		if err != nil {
			return err
		}
		defer f.Close()
		t.cmd.Stderr = f
	} else {
		t.cmd.Stderr = os.Stderr
	}
	t.cmd.Env = env

	// Set SysProcAttr
	t.setSysProcAttr()

	// Run
	err = t.cmd.Run()
	if err != nil {
		return err
	}
	return nil
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
	return t.serialize(false).String()
}

func (t *Task) PrettyString() string {
	return t.serialize(true).String()
}

func (t *Task) serialize(indent bool) *bytes.Buffer {
	t.mu.RLock()
	defer t.mu.RUnlock()
	b := bytes.Buffer{}
	e := json.NewEncoder(&b)
	if indent {
		e.SetIndent("", "  ")
	}
	err := e.Encode(t)
	if err != nil {
		panic(err)
	}
	return &b
}
