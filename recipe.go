package recipe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/DisposaBoy/JsonConfigReader"
	"github.com/pelletier/go-toml"
	"os"
	"path/filepath"
	"sync"
)

type Recipe struct {
	Main   string            `json:"main"`
	Env    map[string]string `json:"env" toml:"env"`
	Interp []string          `json:"interp" toml:"interp"`
	Tasks  map[string]*Task  `json:"tasks"`
	logger *Logger
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

func Open(path string, logger *Logger) (*Recipe, error) {
	var r Recipe
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("(%s) %s", path, err.Error())
	}

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
	r.logger = logger
	r.logger.Debug("Recipe: %s", r.PrettyString())
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
	t.SetEnabled()
	r.logger.Debug("Enabled: %s", name)
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
	for _, t := range r.Tasks {
		if t.IsEnabled() {
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
		nt.t.MustSetRunning()
		ctx, cancel := context.WithCancel(context.Background())
		nt.t.SetCancel(cancel)
		r.logger.Debug("Running: %s", nt.n)
		err := nt.t.Execute(ctx, r)
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
			t.MustSetWaiting()
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
			r.logger.Debug("Failure: %s", result.n)
			// Cancel all the running tasks
			r.onFailure(result.n)
			// Terminate dispatcher
			dispatchAgainCh <- false
			// Terminate
			doneCh <- (*Error)(result)
			break
		}
		r.logger.Debug("Success: %s", result.n)
		if result.n == r.Main {
			dispatchAgainCh <- false
			doneCh <- nil
			break
		}
	success:
		r.onSuccess(result.n)
		dispatchAgainCh <- true
	}
	//r.logger.Debug("Stopping validator")
}

func (r *Recipe) onSuccess(name string) {
	r.Tasks[name].MustSetSuccess()
}

func (r *Recipe) onFailure(name string) {
	for n, t := range r.Tasks {
		if n != name {
			if t.IsRunning() {
				cancel := t.Cancel()
				cancel()
			}
		} else {
			t.MustSetFailure()
		}
	}
}

func (r *Recipe) readyTasks() *TaskIterator {
	namedTasks := make([]*namedTask, 0)
	for n, t := range r.Tasks {
		if r.readyTask(t) {
			namedTasks = append(namedTasks, &namedTask{n, t})
		}
	}
	return &TaskIterator{namedTasks, -1}
}

func (r *Recipe) readyTask(t *Task) bool {
	if !t.IsEnabled() {
		return false
	}
	for _, d := range t.Deps {
		if !r.Tasks[d].IsSuccess() {
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
	return r.string(false)
}

func (r *Recipe) PrettyString() string {
	return r.string(true)
}

func (r *Recipe) string(indent bool) string {
	b := bytes.Buffer{}
	e := json.NewEncoder(&b)
	if indent {
		e.SetIndent("", " ")
	}
	err := e.Encode(r)
	if err != nil {
		panic(err)
	}
	return b.String()
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
