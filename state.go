//go:generate stringer -type=TaskState

package recipe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
)

/*
 * Sources:
 * - https://blog.kowalczyk.info/article/wOYk/advanced-command-execution-in-go-with-osexec.html
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

type State struct {
	States map[string]TaskState `json:"states" toml:"states"`
	path   string
	mu     sync.RWMutex
}

func NewState(path string) *State {
	return &State{
		States: make(map[string]TaskState),
		path:   path,
	}
}

func (s *State) Save() error {
	b := s.serialize(true)
	return ioutil.WriteFile(s.path, b.Bytes(), 0644)
}

func (s *State) Remove() error {
	return os.Remove(s.path)
}

func (s *State) String() string {
	return s.serialize(false).String()
}

func (s *State) PrettyString() string {
	return s.serialize(true).String()
}

func (s *State) serialize(indent bool) *bytes.Buffer {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := bytes.Buffer{}
	e := json.NewEncoder(&b)
	if indent {
		e.SetIndent("", "  ")
	}
	err := e.Encode(s)
	if err != nil {
		panic(err)
	}
	return &b
}

func (s *State) SetDisabled(taskName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.States[taskName] = Disabled
}

func (s *State) SetEnabled(taskName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.States[taskName] != Disabled {
		return fmt.Errorf("Current state must be Disabled, not %d", s.States[taskName])
	}
	s.States[taskName] = Enabled
	return nil
}

func (s *State) MustSetEnabled(taskName string) {
	err := s.SetEnabled(taskName)
	if err != nil {
		panic(err)
	}
}

func (s *State) IsEnabled(taskName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.States[taskName] == Enabled
}

func (s *State) MustSetWaiting(taskName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.States[taskName] != Enabled {
		panic(fmt.Errorf("Current state must be Enabled, not %d", s.States[taskName]))
	}
	s.States[taskName] = Waiting
}

func (s *State) IsWaiting(taskName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.States[taskName] == Waiting
}

func (s *State) MustSetRunning(taskName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.States[taskName] != Waiting {
		panic(fmt.Errorf("Current state must be Waiting, not %d", s.States[taskName]))
	}
	s.States[taskName] = Running
}

func (s *State) IsRunning(taskName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.States[taskName] == Running
}

func (s *State) MustSetSuccess(taskName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.States[taskName] != Running {
		panic(fmt.Errorf("Current state must be Running, not %d", s.States[taskName]))
	}
	s.States[taskName] = Success
}

func (s *State) IsSuccess(taskName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.States[taskName] == Success
}

func (s *State) MustSetFailure(taskName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.States[taskName] != Running {
		panic(fmt.Errorf("Current state must be Running, not %d", s.States[taskName]))
	}
	s.States[taskName] = Failure
}

func (s *State) IsFailure(taskName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.States[taskName] == Failure
}
