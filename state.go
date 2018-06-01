//go:generate stringer -type=TaskState

package recipe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/DisposaBoy/JsonConfigReader"
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
	Cancelled
	Success
	Failure
)

type State struct {
	States map[string]TaskState `json:"states" toml:"states"`
	path   string
	logger *Logger
	mu     sync.RWMutex
}

func OpenState(path string, logger *Logger) (*State, error) {
	var s State

	/* Try to open file */
	f, err := os.Open(path)
	if err != nil {
		/* If is not possible, create and empty struct */
		s = State{
			States: make(map[string]TaskState),
		}
		logger.Info("Creating state file: %s", path)
	} else {
		err = json.NewDecoder(JsonConfigReader.New(f)).Decode(&s)
		if err != nil {
			return nil, fmt.Errorf("(%s) %s", path, err.Error())
		}
		logger.Info("Loading state file: %s", path)
	}
	s.path = path
	s.logger = logger
	return &s, nil
}

func (s *State) Save() error {
	b := s.serialize(true)
	err := ioutil.WriteFile(s.path, b.Bytes(), 0644)
	if err != nil {
		return err
	}
	s.logger.Debug("Saving state file: %s", s.path)
	return nil
}

func (s *State) Remove() error {
	err := os.Remove(s.path)
	if err != nil {
		return err
	}
	s.logger.Info("Removing state file: %s", s.path)
	return nil
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
		return fmt.Errorf("Current state must be Disabled, not %s", s.States[taskName].String())
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
		panic(fmt.Errorf("Current state must be Enabled, not %s", s.States[taskName].String()))
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
		panic(fmt.Errorf("Current state must be Waiting, not %s", s.States[taskName].String()))
	}
	s.States[taskName] = Running
}

func (s *State) IsRunning(taskName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.States[taskName] == Running
}

func (s *State) MustSetCancelled(taskName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.States[taskName] != Running {
		panic(fmt.Errorf("Current state must be Running, not %s", s.States[taskName].String()))
	}
	s.States[taskName] = Cancelled
}

func (s *State) IsCancelled(taskName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.States[taskName] == Cancelled
}

func (s *State) MustSetSuccess(taskName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.States[taskName] != Running {
		panic(fmt.Errorf("Current state must be Running, not %s", s.States[taskName].String()))
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
		panic(fmt.Errorf("Current state must be Running, not %s", s.States[taskName].String()))
	}
	s.States[taskName] = Failure
}

func (s *State) IsFailure(taskName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.States[taskName] == Failure
}

func (s *State) IsDone(taskName string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.States[taskName] == Failure || s.States[taskName] == Success
}
