package recipe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/DisposaBoy/JsonConfigReader"
	"github.com/pelletier/go-toml"
)

type Recipe struct {
	Main   string            `json:"main"`
	Env    map[string]string `json:"env" toml:"env"`
	Interp []string          `json:"interp" toml:"interp"`
	Tasks  map[string]*Task  `json:"tasks"`
	logger *Logger
	state  *State
	mu     sync.RWMutex
}

type namedTask struct {
	n string
	t *Task
}

type result struct {
	n string
	e error
}

func Open(path string, recipeLogger, stateLogger *Logger) (*Recipe, error) {
	var r Recipe
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("(%s) %s", path, err.Error())
	}

	/* Guess type of file and decode it */
	ext := filepath.Ext(path)
	if ext == ".json" {
		err = json.NewDecoder(JsonConfigReader.New(f)).Decode(&r)
		if err != nil {
			return nil, fmt.Errorf("(%s) %s", path, err.Error())
		}
	} else if ext == ".toml" {
		err = toml.NewDecoder(f).Decode(&r)
		if err != nil {
			return nil, fmt.Errorf("(%s) %s", path, err.Error())
		}
	} else {
		// TODO: Implement a modeline mechanism? // -*- coding: utf-8; mode: json; -*-
		return nil, fmt.Errorf("(%s) Unknown filetype", path)
	}

	// NOTE: Create the rest of Recipe fields after the decoding step

	/* Open state */
	r.state, err = OpenState(path+".state", stateLogger)
	if err != nil {
		return nil, err
	}

	/* Set logger */
	r.logger = recipeLogger
	r.logger.Debug("Recipe: %s", r.PrettyString())

	/* Check the recipe */
	if err := r.check(); err != nil {
		return nil, fmt.Errorf("(%s) %s", path, err.Error())
	}
	return &r, nil
}

func (r *Recipe) check() error {
	if len(r.Tasks) <= 0 {
		r.logger.Warning("Empty list of tasks")
		return nil
	}
	for n, t := range r.Tasks {
		if t.Cmd == "" {
			r.logger.Warning("In task '%s': No cmd", n)
		}
		for _, d := range t.Deps {
			if _, ok := r.Tasks[d]; !ok {
				return fmt.Errorf("In task '%s': Unknown referenced task: %s", n, d)
			}
		}
	}
	if r.Main == "" {
		r.logger.Warning("No main task")
	}
	if _, ok := r.Tasks[r.Main]; !ok {
		return fmt.Errorf("Unknown referenced main task: %s", r.Main)
	}
	return nil
}

func (r *Recipe) RunMain(numWorkers uint) error {
	return r.run(numWorkers)
}

func (r *Recipe) RunTask(task string, numWorkers uint) error {
	r.Main = task
	return r.run(numWorkers)
}

func (r *Recipe) enableTasks(name string) error {
	t, ok := r.Tasks[name]
	if !ok {
		return fmt.Errorf("The task is not defined in the recipe: %s", name)
	}
	if !r.state.IsDone(name) {
		r.state.SetDisabled(name)
		r.state.MustSetEnabled(name)
		r.logger.Debug("Enabled: %s", name)
	} else {
		r.logger.Debug("Not enabled: %s", name)
	}
	for _, n := range t.Deps {
		err := r.enableTasks(n)
		if err != nil {
			return nil
		}
	}
	return nil
}

func (r *Recipe) countEnabled() int {
	i := 0
	for n := range r.Tasks {
		if r.state.IsEnabled(n) {
			i++
		}
	}
	return i
}

func (r *Recipe) run(numWorkers uint) error {
	r.logger.Info("Main: %s", r.Main)
	r.logger.Info("Workers: %d", numWorkers)
	err := r.enableTasks(r.Main)
	if err != nil {
		return err
	}
	resultCh := make(chan *result, numWorkers)
	namedTaskCh := make(chan *namedTask, r.countEnabled())
	doneCh := make(chan error)
	dispatchAgainCh := make(chan bool)
	go r.producer(namedTaskCh, dispatchAgainCh)
	for i := uint(0); i < numWorkers; i++ {
		go r.consumer(i, resultCh, namedTaskCh)
	}
	go r.validator(resultCh, dispatchAgainCh, doneCh)
	return <-doneCh
}

func (r *Recipe) consumer(id uint, resultCh chan<- *result, namedTaskCh <-chan *namedTask) {
	//r.logger.Debug("Starting consumer %d", id)
	for nt := range namedTaskCh {
		r.state.MustSetRunning(nt.n)
		r.logger.Debug("Running: %s", nt.n)
		err := nt.t.Execute(r)
		resultCh <- &result{nt.n, err}
	}
	//r.logger.Debug("Stopping consumer %d", id)
}

func (r *Recipe) producer(namedTaskCh chan<- *namedTask, dispatchAgainCh <-chan bool) {
	//r.logger.Debug("Starting producer")
	for {
		r.logger.Debug("Searching ready tasks")
		it := r.readyTasks()
		for n, t := it.next(); t != nil; n, t = it.next() {
			r.state.MustSetWaiting(n)
			r.logger.Debug("Waiting: %s", n)
			namedTaskCh <- &namedTask{n, t}
		}
		if !<-dispatchAgainCh {
			//r.logger.Debug("Stopping producer")
			close(namedTaskCh)
			return
		}
	}
}

func (r *Recipe) validator(resultCh <-chan *result, dispatchAgainCh chan<- bool, doneCh chan<- error) {
	//r.logger.Debug("Starting validator")
	for {
		result := <-resultCh
		if result.e != nil {
			if r.Tasks[result.n].AllowFailure {
				r.logger.Info("Allowed Failure: %s", result.n)
				goto success
			}
			if r.state.IsCancelled(result.n) {
				r.logger.Debug("Cancellation confirmed: %s", result.n)
				goto save
			}
			r.logger.Debug("Failure: %s", result.n)
			// Cancel all the running tasks
			r.onFailure(result.n)
			// Terminate dispatcher
			dispatchAgainCh <- false
			// Terminate
			doneCh <- (*Error)(result)
			goto save
		}
		r.logger.Debug("Success: %s", result.n)
		if result.n == r.Main {
			r.onSuccess(result.n)
			/* Remove the state file if all the tasks have terminated correctly */
			r.state.Remove()
			dispatchAgainCh <- false
			doneCh <- nil
			break
		}
	success:
		r.onSuccess(result.n)
		dispatchAgainCh <- true
	save:
		/* Save the state after any terminated task */
		r.state.Save()
	}
	//r.logger.Debug("Stopping validator")
}

func (r *Recipe) onSuccess(name string) {
	r.state.MustSetSuccess(name)
}

func (r *Recipe) onFailure(name string) {
	for n, t := range r.Tasks {
		if n != name {
			if r.state.IsRunning(n) {
				r.logger.Debug("Cancellation requested: %s", n)
				r.state.MustSetCancelled(n)
				err := t.Terminate()
				if err != nil {
					r.logger.Error("Unable to terminate '%s': %s", n, err.Error())
				}
			}
		} else {
			r.state.MustSetFailure(n)
		}
	}
}

func (r *Recipe) readyTasks() *TaskIterator {
	namedTasks := make([]*namedTask, 0)
	for n, t := range r.Tasks {
		if r.readyTask(n, t) {
			r.logger.Debug("Ready: %s", n)
			namedTasks = append(namedTasks, &namedTask{n, t})
		}
	}
	return &TaskIterator{namedTasks, -1}
}

func (r *Recipe) readyTask(n string, t *Task) bool {
	if !r.state.IsEnabled(n) {
		return false
	}
	for _, d := range t.Deps {
		if !r.state.IsSuccess(d) {
			return false
		}
	}
	return true
}

func (r *Recipe) Environ() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Env
}

func (r *Recipe) Interpreter() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Interp
}

func (r *Recipe) String() string {
	return r.serialize(false).String()
}

func (r *Recipe) PrettyString() string {
	return r.serialize(true).String()
}

func (r *Recipe) serialize(indent bool) *bytes.Buffer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b := bytes.Buffer{}
	e := json.NewEncoder(&b)
	if indent {
		e.SetIndent("", "  ")
	}
	err := e.Encode(r)
	if err != nil {
		panic(err)
	}
	return &b
}

/*
 * Task Iterator
 */

type TaskIterator struct {
	namedTasks []*namedTask
	current    int
}

func (it *TaskIterator) next() (string, *Task) {
	it.current++
	if it.current >= 0 && it.current < len(it.namedTasks) {
		nt := it.namedTasks[it.current]
		return nt.n, nt.t
	}
	return "", nil
}

/*
 * Error
 */

type Error struct {
	n string
	e error
}

func (e *Error) Error() string {
	return fmt.Sprintf("(%s) %s", e.n, e.e.Error())
}
