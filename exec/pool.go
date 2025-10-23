package exec

import (
	"fmt"
	"strings"
	"sync"

	"github.com/netcracker/qubership-core-lib-go/v3/logging"
)

var logger = logging.GetLogger("exec")

type Pool[T any] interface {
	Submit(tasks []Task[T]) (*TaskResults[T], error)
	Stop()
}

type Task[T any] interface {
	Run() (T, error)
}

type TaskResult[T any] struct {
	Result T
	Err    error
}

type TaskResults[T any] struct {
	Results []*TaskResult[T]
}

func (trs *TaskResults[T]) GetResults() []T {
	var results []T
	for _, tr := range trs.Results {
		results = append(results, tr.Result)
	}
	return results
}

func (trs *TaskResults[T]) GetErrors() []error {
	var results []error
	for _, tr := range trs.Results {
		if tr.Err != nil {
			results = append(results, tr.Err)
		}
	}
	return results
}

func (trs *TaskResults[T]) HasErrors() bool {
	return len(trs.GetErrors()) > 0
}

func (trs *TaskResults[T]) GetAsError() error {
	errs := trs.GetErrors()
	var errsStr []string
	for _, errS := range errs {
		errsStr = append(errsStr, errS.Error())
	}
	if len(errsStr) > 0 {
		return fmt.Errorf("%d task(s) failed, errors: \n%s", len(errs), strings.Join(errsStr, "\n"))
	} else {
		return nil
	}
}

func newTaskResults[T any](tResults []*TaskResult[T]) *TaskResults[T] {
	var trs TaskResults[T]
	trs.Results = tResults
	return &trs
}

type taskWrapper[T any] struct {
	Task       Task[T]
	ErrChan    chan error
	ResultChan chan *TaskResult[T]
	wg         *sync.WaitGroup
}

func NewFixedPool[T any](parallelism int, bufferSize int) Pool[T] {
	wChan := make(chan *taskWrapper[T], bufferSize)
	logger.Info("starting %d workers", parallelism)
	for i := 0; i < parallelism; i++ {
		go worker(i+1, wChan)
	}
	pool := fixedSizePool[T]{
		Parallelism: parallelism,
		BufferSize:  bufferSize,
		wChan:       wChan,
		lock:        &sync.RWMutex{},
	}
	return &pool
}

type fixedSizePool[T any] struct {
	Parallelism int
	BufferSize  int
	wChan       chan *taskWrapper[T]
	stopped     bool
	lock        *sync.RWMutex
}

func (p *fixedSizePool[T]) Submit(tasks []Task[T]) (*TaskResults[T], error) {
	p.lock.RLock()
	tasksAmount := len(tasks)
	logger.Debug("submitting %d tasks", tasksAmount)
	defer p.lock.RUnlock()
	if p.stopped {
		return nil, fmt.Errorf("pool is stopped")
	}
	var wg sync.WaitGroup
	for i := 0; i < tasksAmount; i++ {
		wg.Add(1)
	}
	resultChan := make(chan *TaskResult[T], tasksAmount)
	go func() {
		for _, tsk := range tasks {
			p.wChan <- &taskWrapper[T]{Task: tsk, ResultChan: resultChan, wg: &wg}
		}
	}()
	var results []*TaskResult[T]
	wg.Wait()
	for i := 0; i < tasksAmount; i++ {
		results = append(results, <-resultChan)
	}
	return newTaskResults(results), nil
}

func (p *fixedSizePool[T]) Stop() {
	p.lock.Lock()
	defer p.lock.Unlock()
	if !p.stopped {
		logger.Debug("stopping pool")
		p.stopped = true
		close(p.wChan)
	}
}

func (p *fixedSizePool[T]) IsRunning() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return !p.stopped
}

func worker[T any](id int, wChan chan *taskWrapper[T]) {
	defer func() {
		if err := recover(); err != nil {
			logger.Error("panic occurred in worker #%d, recovering: %+v", id, err)
			worker(id, wChan)
		}
	}()
	for task := range wChan {
		result, err := task.Task.Run()
		tr := TaskResult[T]{Result: result, Err: err}
		task.ResultChan <- &tr
		task.wg.Done()
	}
	logger.Debug("worker #%d has finished", id)
}
